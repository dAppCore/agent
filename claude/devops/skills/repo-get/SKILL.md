---
name: repo-get
description: This skill should be used when the user asks to "get repo info", "show repo", "repo details", or needs to see details about a specific Forge repository including default branch, visibility, and archive status.
argument-hint: <repo> [--org=core]
allowed-tools: ["Bash"]
---

# Get Forge Repository Info

Fetch and display repository details from Forge.

```bash
core-agent repo/get <repo> [--org=core]
```

Example:
```bash
core-agent repo/get go
core-agent repo/get agent
```
