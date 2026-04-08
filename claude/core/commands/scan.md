---
name: scan
description: Scan Forge repos for open issues with actionable labels (agentic, help-wanted, bug)
arguments:
  - name: org
    description: Forge org to scan
    default: core
---

Use the `mcp__plugin_agent_agent__agentic_scan` tool with org: $ARGUMENTS.org

Show results as a table with columns: Repo, Issue #, Title, Labels.
