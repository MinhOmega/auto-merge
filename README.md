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
        uses: MinhOmega/auto-merge@v1
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

## Example

This workflow automates the merging of pull requests after they have received the required approvals. To use this action, ensure you have the necessary permissions and secrets set up in your repository.

1. Create a new personal access token with `write:packages`, `read:packages`, and `delete:packages` scopes.
2. Add this token to your repository secrets as `GITHUB_TOKEN`.

Here's an example of a workflow file:

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
        uses: MinhOmega/auto-merge@v1
        with:
          github_token: ${{ secrets.GITHUB_TOKEN }}
```

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

