---
name: workspace-clean
description: This skill should be used when the user asks to "clean workspaces", "clean up agents", "remove stale workspaces", "nuke completed", or needs to remove finished/failed/blocked agent workspaces.
argument-hint: [all|completed|failed|blocked]
allowed-tools: ["Bash"]
---

# Clean Agent Workspaces

Remove stale agent workspaces. Never removes running workspaces.

```bash
# Remove all non-running workspaces
core-agent workspace/clean all

# Remove only completed/merged
core-agent workspace/clean completed

# Remove only failed
core-agent workspace/clean failed

# Remove only blocked
core-agent workspace/clean blocked
```
