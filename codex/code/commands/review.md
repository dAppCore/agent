---
name: review
description: Perform a code review on staged changes, a commit range, or a GitHub PR
args: <range> [--security]
---

# Code Review

Performs a code review on the specified changes.

## Usage

Review staged changes:
`/code:review`

Review a commit range:
`/code:review HEAD~3..HEAD`

Review a GitHub PR:
`/code:review #123`

Perform a security-focused review:
`/code:review --security`

## Action

```bash
"${CLAUDE_PLUGIN_ROOT}/scripts/code-review.sh" "$@"
```
