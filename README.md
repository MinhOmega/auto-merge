# Auto Merge PRs

A GitHub Action to automatically merge pull requests after they have received the required approvals.

## Usage

To use this action, create a workflow YAML file in your .github/workflows directory in your GitHub repository. Here is an example:

```yaml
name: Auto-Merge-PRs

on:
  pull_request:
    types: [opened, reopened, synchronize]

jobs:
  auto-merge:
    runs-on: ubuntu-latest

    steps:
      - name: Checkout repository
        uses: actions/checkout@v2

      - name: Run Auto-Merge PRs
        uses: MinhOmega/auto-merge@v1.0.1
        with:
          github_token: ${{ secrets.GITHUB_TOKEN }}
```

## Inputs

| Input            | Description                                        | Required | Default |
|------------------|----------------------------------------------------|----------|---------|
| `sleep_duration` | Duration to sleep between checks (in seconds)      | false    | `5`     |
| `timeout_minutes`| Timeout for the merge process (in minutes)         | false    | `1440`  |
| `base_branch`    | Base branch to check against                       | false    | `master`|
| `github_token`   | GitHub token for authentication                    | true     | N/A     |

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

