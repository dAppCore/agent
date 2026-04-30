---
name: status
description: Show status of all agent workspaces (running, completed, blocked, failed)
---

Use the `mcp__plugin_agent_agent__agentic_status` tool to list all agent workspaces.

Show results as a table with columns: Name, Status, Agent, Repo, Task, Age.

For blocked workspaces, show the question from BLOCKED.md.
For completed workspaces with output, show the last 10 lines of the agent log.
