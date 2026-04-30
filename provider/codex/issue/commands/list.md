---
name: list
description: List open issues
hooks:
  PreToolUse:
    - hooks:
        - type: command
          command: "${CLAUDE_PLUGIN_ROOT}/scripts/list.sh"
---

# List open issues
