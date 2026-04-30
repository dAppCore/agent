---
name: workspace-list
description: This skill should be used when the user asks to "list workspaces", "show agents", "what's running", "workspace status", "active agents", or wants to see the current state of all agent workspaces.
argument-hint: (no arguments needed)
allowed-tools: ["Bash"]
---

# List Agent Workspaces

Show all agent workspaces with their status, agent type, and repo.

```bash
core-agent workspace/list
```

Output shows: status, agent, repo, workspace name. Statuses: running, completed, failed, blocked, merged, queued.
