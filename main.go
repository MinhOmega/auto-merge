package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

type PullRequestEvent struct {
	Number int  `json:"number"`
	Base   Base `json:"base"`
	User   User `json:"user"`
}

type Base struct {
	Ref string `json:"ref"`
}

type User struct {
	Login string `json:"login"`
}

func main() {
	// Load environment variables
	sleepDuration, err := strconv.Atoi(getEnv("SLEEP_DURATION", "5"))
	if err != nil {
		fmt.Printf("Error parsing SLEEP_DURATION: %v\n", err)
		os.Exit(1)
	}

	timeoutMinutes, err := strconv.Atoi(getEnv("TIMEOUT_MINUTES", "1440"))
	if err != nil {
		fmt.Printf("Error parsing TIMEOUT_MINUTES: %v\n", err)
		os.Exit(1)
	}

	baseBranch := getEnv("BASE_BRANCH", "master")
	githubToken := os.Getenv("GITHUB_TOKEN")

	if githubToken == "" {
		fmt.Println("GITHUB_TOKEN is not set")
		os.Exit(1)
	}

	// Load GitHub event data
	eventPath := os.Getenv("GITHUB_EVENT_PATH")
	eventData := loadPullRequestEvent(eventPath)

	// Authenticate GitHub CLI
	if _, err := execCommand("gh", "auth", "login", "--with-token", githubToken); err != nil {
		fmt.Printf("Error authenticating GitHub CLI: %v\n", err)
		os.Exit(1)
	}

	// Get the pull request details
	prNumber := eventData.Number
	prBaseBranch := eventData.Base.Ref
	prAuthor := eventData.User.Login

	endTime := time.Now().Add(time.Duration(timeoutMinutes) * time.Minute)
	commented := false

	var reviewers string

	for {
		prState, err := execCommand("gh", "pr", "view", strconv.Itoa(prNumber), "--json", "state", "--jq", ".state")
		if err != nil {
			fmt.Printf("Error getting PR state: %v\n", err)
			os.Exit(1)
		}

		if prState == "CLOSED" {
			fmt.Printf("🛑 PR #%d is closed. Stopping the process.\n", prNumber)
			return
		}

		reviewers, err = execCommand("gh", "pr", "view", strconv.Itoa(prNumber), "--json", "reviewRequests", "--jq", ".reviewRequests[].login")
		if err != nil {
			fmt.Printf("Error getting PR reviewers: %v\n", err)
			os.Exit(1)
		}

		if reviewers == "" {
			if !commented {
				if _, err := execCommand("gh", "pr", "comment", strconv.Itoa(prNumber), "--body", fmt.Sprintf("Hi @%s, the pull request needs to be assigned to someone for review and approval. Please assign reviewers. Thank you!", prAuthor)); err != nil {
					fmt.Printf("Error commenting on PR: %v\n", err)
					os.Exit(1)
				}
				commented = true
			}
			fmt.Println("Waiting for reviewers to be assigned...")
			time.Sleep(time.Duration(sleepDuration) * time.Second)
		} else {
			break
		}
	}

	if prBaseBranch == baseBranch {
		reviewersList := strings.Fields(reviewers)
		numReviewers := len(reviewersList)
		requiredApprovals := numReviewers/2 + 1

		for time.Now().Before(endTime) {
			prState, err := execCommand("gh", "pr", "view", strconv.Itoa(prNumber), "--json", "state", "--jq", ".state")
			if err != nil {
				fmt.Printf("Error getting PR state: %v\n", err)
				os.Exit(1)
			}

			if prState == "CLOSED" {
				fmt.Printf("🛑 PR #%d is closed. Stopping the process.\n", prNumber)
				return
			}

			approvedCount := 0
			approvedUsers := []string{}

			for _, reviewer := range reviewersList {
				approvalCheck, err := execCommand("gh", "pr", "view", strconv.Itoa(prNumber), "--json", "reviews", "--jq", fmt.Sprintf(".reviews[] | select(.author.login==\"%s\" and .state==\"APPROVED\")", reviewer))
				if err != nil {
					fmt.Printf("Error checking approval for reviewer %s: %v\n", reviewer, err)
					os.Exit(1)
				}

				if strings.Contains(approvalCheck, reviewer) {
					approvedCount++
					approvedUsers = append(approvedUsers, fmt.Sprintf("@%s", reviewer))
				}
			}

			if approvedCount >= requiredApprovals {
				if _, err := execCommand("gh", "pr", "merge", strconv.Itoa(prNumber), "--merge", "--repo", os.Getenv("GITHUB_REPOSITORY"), "--admin", "--body", "This PR was merged by the GitHub Actions bot."); err != nil {
					fmt.Printf("Error merging PR: %v\n", err)
					os.Exit(1)
				}

				comment := fmt.Sprintf("💬 This Pull Request is auto-merged by approval of %s 🗨️", strings.Join(approvedUsers, " "))
				if _, err := execCommand("gh", "pr", "comment", strconv.Itoa(prNumber), "--body", comment); err != nil {
					fmt.Printf("Error commenting on PR: %v\n", err)
					os.Exit(1)
				}

				if _, err := execCommand("gh", "label", "create", "auto-merge", "--color", "0e8a16"); err != nil {
					fmt.Printf("Error creating label: %v\n", err)
				}

				if _, err := execCommand("gh", "pr", "edit", strconv.Itoa(prNumber), "--add-label", "auto-merge"); err != nil {
					fmt.Printf("Error adding label to PR: %v\n", err)
				}

				return
			} else {
				fmt.Printf("🔄 PR #%d does not have the required approvals yet. Checking again in %d seconds...\n", prNumber, sleepDuration)
				time.Sleep(time.Duration(sleepDuration) * time.Second)
			}
		}

		fmt.Printf("🕰️ PR #%d did not receive the required approvals within the timeout period.\n", prNumber)
	} else {
		fmt.Printf("❌ PR does not target the '%s' branch. No merge action will be taken.\n", baseBranch)
	}
}

func getEnv(key, defaultValue string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return defaultValue
}

func loadPullRequestEvent(path string) PullRequestEvent {
	file, err := os.ReadFile(path)
	if err != nil {
		fmt.Printf("Error reading event file: %v\n", err)
		os.Exit(1)
	}

	var eventData PullRequestEvent
	err = json.Unmarshal(file, &eventData)
	if err != nil {
		fmt.Printf("Error unmarshalling event data: %v\n", err)
		os.Exit(1)
	}

	return eventData
}

func execCommand(name string, arg ...string) (string, error) {
	cmd := exec.Command(name, arg...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("%s: %v", output, err)
	}
	return strings.TrimSpace(string(output)), nil
}
