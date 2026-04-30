---
name: pr-list
description: This skill should be used when the user asks to "list PRs", "show pull requests", "what PRs are open", "pending PRs", or needs to see pull requests for a Forge repo.
argument-hint: <repo> [--org=core]
allowed-tools: ["Bash"]
---

# List Forge Pull Requests

List all pull requests for a Forge repository. Shows state, branches, title.

```bash
core-agent pr/list <repo> [--org=core]
```

Example:
```bash
core-agent pr/list go
core-agent pr/list agent
```
