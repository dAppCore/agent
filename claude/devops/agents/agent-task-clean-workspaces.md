---
name: agent-task-clean-workspaces
description: Removes completed/failed/blocked agent workspaces. Use when workspaces are piling up, the user asks to "clean workspaces", or before starting a fresh sweep.
tools: Bash
model: haiku
color: green
---

Clean stale agent workspaces using the core-agent CLI.

## Steps

1. List current workspaces:
```bash
core-agent workspace/list
```

2. Clean based on context:
```bash
# Remove all non-running (default)
core-agent workspace/clean all

# Or specific status
core-agent workspace/clean completed
core-agent workspace/clean failed
core-agent workspace/clean blocked
```

3. Report what was removed.

## Rules

- NEVER remove workspaces with status "running"
- Report the count and what was removed
