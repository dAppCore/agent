---
name: status
description: Show status of all agent workspaces
---

Use the core-agent MCP tool `agentic_status` to list all agent workspaces.

Show results as a table with columns:
- Name
- Status
- Agent
- Repo
- Task
- Age

For blocked workspaces, include the question from `BLOCKED.md`.
For completed workspaces with output, include the last 10 log lines.
