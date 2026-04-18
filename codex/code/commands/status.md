---
name: status
description: Show status across all Host UK repos
args: [--dirty|--behind]
---

# Multi-Repo Status

Wraps `core dev health` with better formatting.
name: /core:status
description: Show status across all Host UK repos
hooks:
  AfterToolConfirmation:
    - hooks:
        - type: command
          command: "${CLAUDE_PLUGIN_ROOT}/scripts/status.sh"
---

# Repo Status

A quick command to show the status across all Host UK repos.

## Usage

`/core:status` - Show all repo statuses
`/core:status --dirty` - Only show repos with changes
`/core:status --behind` - Only show repos behind remote

## Action

Run this command to get the status:

```bash
"${CLAUDE_PLUGIN_ROOT}/scripts/core-status.sh" "$@"
```
