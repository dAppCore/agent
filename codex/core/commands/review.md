---
name: review
description: Review completed agent workspace and show merge options
arguments:
  - name: workspace
    description: Workspace name (e.g. go-html-1773592564). If omitted, shows all completed.
---

If no workspace is specified, use the core-agent MCP tool `agentic_status` to list all workspaces, then show only completed ones with a summary table.

If a workspace is specified:
1. Read the agent log file: `.core/workspace/{workspace}/agent-*.log`
2. Show the last 30 lines of output
3. Check git history in the workspace: `git -C .core/workspace/{workspace}/src log --oneline main..HEAD`
4. Show the diff stat: `git -C .core/workspace/{workspace}/src diff --stat main`
5. Offer next actions:
   - Merge
   - Discard
   - Resume
