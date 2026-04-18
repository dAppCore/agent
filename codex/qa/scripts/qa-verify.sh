#!/bin/bash
# Verify QA passes before stopping during /core:qa mode
#
# Stop hook that runs QA checks and blocks if any failures exist.
# Ensures Claude fixes all issues before completing the task.

read -r input
STOP_ACTIVE=$(echo "$input" | jq -r '.stop_hook_active // false')

# Prevent infinite loop
if [ "$STOP_ACTIVE" = "true" ]; then
    exit 0
fi

# Detect project type and run QA
if [ -f "go.mod" ]; then
    PROJECT="go"
    RESULT=$(core go qa 2>&1) || true
elif [ -f "composer.json" ]; then
    PROJECT="php"
    RESULT=$(core php qa 2>&1) || true
else
    # Not a Go or PHP project, allow stop
    exit 0
fi

# Check if QA passed
if echo "$RESULT" | grep -qE "FAIL|ERROR|✗|panic:|undefined:"; then
    # Extract top issues for context
    ISSUES=$(echo "$RESULT" | grep -E "^(FAIL|ERROR|✗|undefined:|panic:)|^[a-zA-Z0-9_/.-]+\.(go|php):[0-9]+:" | head -5)

    # Escape for JSON
    ISSUES_ESCAPED=$(echo "$ISSUES" | sed 's/\\/\\\\/g' | sed 's/"/\\"/g' | sed ':a;N;$!ba;s/\n/\\n/g')

    cat << EOF
{
  "decision": "block",
  "reason": "QA still has issues:\n\n$ISSUES_ESCAPED\n\nPlease fix these before stopping."
}
EOF
else
    # QA passed, allow stop
    exit 0
fi
