---
name: issue-list
description: This skill should be used when the user asks to "list issues", "show issues", "what issues are open", or needs to see issues for a Forge repo.
argument-hint: <repo> [--org=core]
allowed-tools: ["Bash"]
---

# List Forge Issues

List all issues for a Forge repository.

```bash
core-agent issue/list <repo> [--org=core]
```

Example:
```bash
core-agent issue/list go
core-agent issue/list agent
```
