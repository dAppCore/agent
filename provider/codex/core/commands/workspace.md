---
name: workspace
description: Manage CoreAgent workspaces, queue state, watches, and permanent dispatch stats
args: "[list|clean|stats|dispatch|watch] [options]"
---

# Workspace Orchestration

Use this command family when the user asks about active agents, queued work, permanent dispatch history, workspace cleanup, or watching work to finish.

## Preferred Routing

Use MCP tools when available:
- `agentic_status` for current workspace status
- `agentic_dispatch` for dispatching a task
- `agentic_watch` for waiting on running or queued workspaces

Use the local CLI fallback when MCP tools are unavailable:

```bash
core-agent workspace list
core-agent workspace stats --limit=20
core-agent workspace dispatch <repo> --task="..." --issue=N|--pr=N|--branch=X
core-agent workspace watch <workspace>
core-agent workspace clean completed
```

## Subcommands

| Subcommand | Purpose |
|------------|---------|
| `list` | Show current workspace status from `status.json` files |
| `stats` | Read permanent dispatch history from `.core/workspace/db.duckdb` |
| `dispatch` | Dispatch an agent with queue and concurrency handling |
| `watch` | Wait for one or more workspaces to complete |
| `clean` | Remove completed, failed, blocked, or all workspaces after recording stats |

## Output

Return compact tables. For `stats`, include workspace, status, agent, duration, findings, and completion time. For `watch`, report only status transitions and final outcome.
