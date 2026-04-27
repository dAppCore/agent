---
name: review
description: Review completed agent workspace — show output, git diff, and merge options
arguments:
  - name: workspace
    description: Workspace name (e.g. go-html-1773592564). If omitted, shows all completed.
---

If no workspace specified, use `mcp__plugin_agent_agent__agentic_status` to list all workspaces, then show only completed ones with a summary table.

If workspace specified:
1. Read the agent log file: `.core/workspace/{workspace}/agent-*.log`
2. Show the last 30 lines of output
3. Check git diff in the workspace: `git -C .core/workspace/{workspace}/src log --oneline main..HEAD`
4. Show the diff stat: `git -C .core/workspace/{workspace}/src diff --stat main`
5. Ask if the user wants to:
   - **Merge**: fetch branch into real repo, push to forge
   - **Discard**: delete the workspace
   - **Resume**: dispatch another agent to continue the work
