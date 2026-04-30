---
name: issue-get
description: This skill should be used when the user asks to "get issue", "show issue", "read issue", "fetch issue", or needs to view a specific Forge issue by number.
argument-hint: <repo> --number=N [--org=core]
allowed-tools: ["Bash"]
---

# Get Forge Issue

Fetch and display a Forge issue by number.

```bash
core-agent issue/get <repo> --number=N [--org=core]
```

Example:
```bash
core-agent issue/get go --number=16
core-agent issue/get agent --number=5 --org=core
```
