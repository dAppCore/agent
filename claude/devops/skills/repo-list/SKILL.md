---
name: repo-list
description: This skill should be used when the user asks to "list repos", "show repos", "what repos exist", "how many repos", or needs to see all repositories in a Forge organisation.
argument-hint: [--org=core]
allowed-tools: ["Bash"]
---

# List Forge Repositories

List all repositories in a Forge organisation.

```bash
core-agent repo/list [--org=core]
```

Example:
```bash
core-agent repo/list
core-agent repo/list --org=lthn
```
