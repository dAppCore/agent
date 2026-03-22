---
name: clean-workspaces
description: This skill should be used when the user asks to "clean workspaces", "clean up old agents", "remove stale workspaces", "nuke completed workspaces", or needs to remove finished/failed agent workspaces from the dispatch queue.
argument-hint: [--all | --completed | --failed | --blocked]
allowed-tools: ["Bash"]
---

# Clean Agent Workspaces

Remove stale agent workspaces from the dispatch system.

## Steps

1. List all workspaces with their status:
   ```bash
   for ws in /Users/snider/Code/.core/workspace/*/status.json; do
     dir=$(dirname "$ws")
     name=$(basename "$dir")
     status=$(python3 -c "import json; print(json.load(open('$ws'))['status'])" 2>/dev/null || echo "unknown")
     echo "$status $name"
   done | sort
   ```

2. Based on the argument:
   - `--completed` — remove workspaces with status "completed"
   - `--failed` — remove workspaces with status "failed"
   - `--blocked` — remove workspaces with status "blocked"
   - `--all` — remove completed + failed + blocked (NOT running)
   - No argument — show the list and ask the user what to remove

3. Show the user what will be removed and get confirmation BEFORE deleting.

4. Remove confirmed workspaces:
   ```bash
   rm -rf /Users/snider/Code/.core/workspace/<name>/
   ```

5. Report how many were removed.

## Important

- NEVER remove workspaces with status "running" — they have active processes
- ALWAYS ask for confirmation before removing
- Show the count and names before removing
