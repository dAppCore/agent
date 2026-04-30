---
name: issue-comment
description: This skill should be used when the user asks to "comment on issue", "add comment", "reply to issue", or needs to post a comment on a Forge issue.
argument-hint: <repo> --number=N --body="comment text" [--org=core]
allowed-tools: ["Bash"]
---

# Comment on Forge Issue

Post a comment on a Forge issue.

```bash
core-agent issue/comment <repo> --number=N --body="comment text" [--org=core]
```

Example:
```bash
core-agent issue/comment go --number=16 --body="Fixed in v0.6.0"
```
