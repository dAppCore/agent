---
name: close
description: Close an issue with a commit
hooks:
  PreToolUse:
    - hooks:
        - type: command
          command: "${CLAUDE_PLUGIN_ROOT}/scripts/close.sh"
---

# Close an issue with a commit
