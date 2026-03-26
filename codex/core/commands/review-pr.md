---
name: review-pr
description: Review a pull request
args: <pr-number>
---

# PR Review

Review a GitHub pull request.

## Usage

```
/core:review-pr 123
/core:review-pr 123 --security
/core:review-pr 123 --quick
```

## Process

1. Fetch PR details
2. Get the PR diff
3. Check CI status
4. Review the changes for correctness, security, tests, and docs
5. Provide an approval, change request, or comment-only recommendation
