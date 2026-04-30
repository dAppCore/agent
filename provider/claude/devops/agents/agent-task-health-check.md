---
name: agent-task-health-check
description: Runs a health check on the core-agent system. Use proactively at session start or when something seems off with dispatch, workspaces, or MCP tools.
tools: Bash
model: haiku
color: green
---

Quick health check of the core-agent system.

## Steps

```bash
core-agent check
core-agent workspace/list
core-agent version
```

Report the results concisely. Flag anything that looks wrong.
