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
	Number      int `json:"number"`
	PullRequest struct {
		Base struct {
			Ref string `json:"ref"`
		} `json:"base"`
		User struct {
			Login string `json:"login"`
		} `json:"user"`
	} `json:"pull_request"`
}

func main() {
	// Read environment variables
	sleepDuration, _ := strconv.Atoi(getEnv("SLEEP_DURATION", "5"))
	timeoutMinutes, _ := strconv.Atoi(getEnv("TIMEOUT_MINUTES", "10"))
	baseBranch := getEnv("BASE_BRANCH", "master")
	ghToken := os.Getenv("GH_TOKEN")

	// Authenticate GitHub CLI
	runCommand("echo", ghToken, "|", "gh", "auth", "login", "--with-token")

	// Read event data
	eventPath := os.Getenv("GITHUB_EVENT_PATH")
	eventFile, err := os.Open(eventPath)
	if err != nil {
		fmt.Printf("Error opening event file: %v\n", err)
		os.Exit(1)
	}
	defer eventFile.Close()

	var event PullRequestEvent
	json.NewDecoder(eventFile).Decode(&event)

	prNumber := event.Number
	prBaseBranch := event.PullRequest.Base.Ref
	prAuthor := event.PullRequest.User.Login

	// Set end time for the merge process
	endTime := time.Now().Add(time.Duration(timeoutMinutes) * time.Minute)

	// Initialize flag for commenting
	commented := false

	var reviewers []string
	for {
		// Check if the pull request is closed or merged
		prState := getPRState(prNumber)
		fmt.Printf("Current PR state: %s\n", prState)

		if prState == "CLOSED" {
			fmt.Printf("🛑 PR #%d is closed. Stopping the process.\n", prNumber)
			os.Exit(0)
		} else if prState == "MERGED" {
			fmt.Printf("✅ PR #%d is merged. Proceeding to the next step.\n", prNumber)
			break
		}

		reviewers = getPRReviewers(prNumber)
		if len(reviewers) == 0 {
			if !commented {
				addPRComment(prNumber, fmt.Sprintf("Hi @%s, the pull request needs to be assigned to someone for review and approval. Please assign reviewers. Thank you!", prAuthor))
				commented = true
			}
			fmt.Println("Waiting for reviewers to be assigned...")
			time.Sleep(time.Duration(sleepDuration) * time.Second)

			// Check again if the PR is closed or merged
			prState = getPRState(prNumber)
			fmt.Printf("Current PR state: %s\n", prState)

			if prState == "CLOSED" {
				fmt.Printf("🛑 PR #%d is closed. Stopping the process.\n", prNumber)
				os.Exit(0)
			} else if prState == "MERGED" {
				fmt.Printf("✅ PR #%d is merged. Proceeding to the next step.\n", prNumber)
				break
			}

			if time.Now().After(endTime) {
				fmt.Println("⏳ Timeout reached while waiting for reviewers. Stopping the process.")
				os.Exit(0)
			}
		} else {
			break
		}
	}

	if prBaseBranch == baseBranch {
		numReviewers := len(reviewers)
		requiredApprovals := (numReviewers / 2) + 1
		approvedCount := 0

		for time.Now().Before(endTime) {
			prState := getPRState(prNumber)
			fmt.Printf("Current PR state: %s\n", prState)

			if prState == "CLOSED" {
				fmt.Printf("🛑 PR #%d is closed. Stopping the process.\n", prNumber)
				os.Exit(0)
			} else if prState == "MERGED" {
				fmt.Printf("✅ PR #%d is merged. Proceeding to the next step.\n", prNumber)
				break
			}

			approvedCount = 0
			approvedUsers := []string{}

			for _, reviewer := range reviewers {
				if isReviewerApproved(prNumber, reviewer) {
					approvedCount++
					approvedUsers = append(approvedUsers, "@"+reviewer)
				}
			}

			if approvedCount >= requiredApprovals {
				authorizedUser := getAuthenticatedUser()
				mergePR(prNumber, authorizedUser, approvedUsers)
				os.Exit(0)
			} else {
				fmt.Printf("🔄 PR #%d does not have the required approvals yet. Checking again in %d seconds...\n", prNumber, sleepDuration)
				time.Sleep(time.Duration(sleepDuration) * time.Second)
			}
		}

		if approvedCount < requiredApprovals {
			fmt.Printf("🕰️ PR #%d did not receive the required approvals within the timeout period.\n", prNumber)
		}
	} else {
		fmt.Printf("❌ PR does not target the '%s' branch. No merge action will be taken.\n", baseBranch)
	}
}

// Helper functions
func getEnv(key, fallback string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return fallback
}

func runCommand(name string, arg ...string) {
	cmd := exec.Command(name, arg...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err := cmd.Run()
	if err != nil {
		fmt.Printf("Command %s failed with %s\n", strings.Join(cmd.Args, " "), err)
		os.Exit(1)
	}
}

func runCommandAndGetOutput(name string, arg ...string) string {
	cmd := exec.Command(name, arg...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Printf("Command %s failed with %s\n", strings.Join(cmd.Args, " "), err)
		os.Exit(1)
	}
	return string(output)
}

func getPRState(prNumber int) string {
	output := runCommandAndGetOutput("gh", "pr", "view", strconv.Itoa(prNumber), "--json", "state", "--jq", ".state")
	return strings.TrimSpace(output)
}

func getPRReviewers(prNumber int) []string {
	output := runCommandAndGetOutput("gh", "pr", "view", strconv.Itoa(prNumber), "--json", "reviewRequests", "--jq", ".reviewRequests[].login")
	return strings.Fields(output)
}

func addPRComment(prNumber int, comment string) {
	runCommand("gh", "pr", "comment", strconv.Itoa(prNumber), "--body", comment)
}

func getAuthenticatedUser() string {
	output := runCommandAndGetOutput("gh", "api", "user", "--jq", ".login")
	return strings.TrimSpace(output)
}

func isReviewerApproved(prNumber int, reviewer string) bool {
	output := runCommandAndGetOutput("gh", "pr", "view", strconv.Itoa(prNumber), "--json", "reviews", "--jq", fmt.Sprintf(".reviews[] | select(.author.login==\"%s\" and .state==\"APPROVED\")", reviewer))
	return strings.TrimSpace(output) != ""
}

func mergePR(prNumber int, authorizedUser string, approvedUsers []string) {
	runCommand("gh", "pr", "merge", strconv.Itoa(prNumber), "--merge", "--repo", os.Getenv("GITHUB_REPOSITORY"), "--admin", "--body", "This PR was merged by the GitHub Actions bot.")
	comment := fmt.Sprintf("💬 This Pull Request is auto-merged by approval of %s 🗨️", strings.Join(approvedUsers, " "))
	addPRComment(prNumber, comment)
	runCommand("gh", "label", "create", "auto-merge", "--color", "0e8a16")
	runCommand("gh", "pr", "edit", strconv.Itoa(prNumber), "--add-label", "auto-merge")
}
