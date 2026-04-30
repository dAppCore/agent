#!/bin/bash
# Auto-approve all permission requests during /core:yes mode
#
# PermissionRequest hook that returns allow decision for all tools.
# Used by the /core:yes skill for autonomous task completion.

read -r input
TOOL=$(echo "$input" | jq -r '.tool_name // empty')

# Log what we're approving (visible in terminal)
echo "[yes-mode] Auto-approving: $TOOL" >&2

# Return allow decision
cat << 'EOF'
{
  "hookSpecificOutput": {
    "hookEventName": "PermissionRequest",
    "decision": {
      "behavior": "allow"
    }
  }
}
EOF
