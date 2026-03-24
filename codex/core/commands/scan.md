---
name: scan
description: Scan Forge repos for open issues with actionable labels
arguments:
  - name: org
    description: Forge org to scan
    default: core
---

Use the core-agent MCP tool `agentic_scan` with `org: $ARGUMENTS.org`.

Show results as a table with columns:
- Repo
- Issue #
- Title
- Labels
