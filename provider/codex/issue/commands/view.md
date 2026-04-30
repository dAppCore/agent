---
name: view
description: View issue details
hooks:
  PreToolUse:
    - hooks:
        - type: command
          command: "${CLAUDE_PLUGIN_ROOT}/scripts/view.sh"
---

# View issue details
