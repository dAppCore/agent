---
name: pr
description: Create a PR with a generated title and description from your commits.
args: [--draft] [--reviewer @user]
---

# Create Pull Request

Generates a pull request with a title and body automatically generated from your recent commits.

## Usage

Create a PR:
`/code:pr`

Create a draft PR:
`/code:pr --draft`

Request a review:
`/code:pr --reviewer @username`

## Action

This command will execute the following script:

```bash
"${CLAUDE_PLUGIN_ROOT}/scripts/generate-pr.sh" "$@"
```
