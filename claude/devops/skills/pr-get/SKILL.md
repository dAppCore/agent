---
name: pr-get
description: This skill should be used when the user asks to "get PR", "show PR", "read pull request", "fetch PR", or needs to view a specific Forge pull request by number.
argument-hint: <repo> --number=N [--org=core]
allowed-tools: ["Bash"]
---

# Get Forge Pull Request

Fetch and display a Forge PR by number. Shows state, branch, mergeability.

```bash
core-agent pr/get <repo> --number=N [--org=core]
```

Example:
```bash
core-agent pr/get go --number=22
```
