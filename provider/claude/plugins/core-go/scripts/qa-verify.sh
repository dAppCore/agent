#!/bin/bash
# Verify QA passes before stopping during /core-go:qa mode.

read -r input
STOP_ACTIVE=$(echo "$input" | jq -r '.stop_hook_active // false')

# Prevent infinite loop
if [ "$STOP_ACTIVE" = "true" ]; then
    exit 0
fi

# Run Go QA for this plugin family.
RESULT=$(core go qa 2>&1) || true

# Check if QA passed
if echo "$RESULT" | grep -qE "FAIL|ERROR|✗|panic:|undefined:"; then
    ISSUES=$(echo "$RESULT" | grep -E "^(FAIL|ERROR|✗|undefined:|panic:)|^[a-zA-Z0-9_/.-]+\.go:[0-9]+:" | head -5)
    ISSUES_ESCAPED=$(echo "$ISSUES" | sed 's/\\/\\\\/g' | sed 's/"/\\"/g' | sed ':a;N;$!ba;s/\n/\\n/g')

    cat << EOF
{
  "decision": "block",
  "reason": "QA still has issues:\n\n$ISSUES_ESCAPED\n\nPlease fix these before stopping."
}
EOF
else
    exit 0
fi
