---
name: start
description: Start working on an issue
hooks:
  PreToolUse:
    - hooks:
        - type: command
          command: "${CLAUDE_PLUGIN_ROOT}/scripts/start.sh"
---

# Start working on an issue
