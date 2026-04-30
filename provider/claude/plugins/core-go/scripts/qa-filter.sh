#!/bin/bash
# Filter QA output to show only actionable issues during /core:qa mode
#
# PostToolUse hook that processes QA command output and extracts
# only the failures, hiding verbose success output.

read -r input
COMMAND=$(echo "$input" | jq -r '.tool_input.command // empty')
OUTPUT=$(echo "$input" | jq -r '.tool_response.stdout // .tool_response.output // empty')
EXIT_CODE=$(echo "$input" | jq -r '.tool_response.exit_code // 0')

# Only process QA-related commands
case "$COMMAND" in
    "core go qa"*|"core php qa"*|"core go test"*|"core php test"*|"core go lint"*|"core php stan"*)
        ;;
    *)
        # Not a QA command, pass through unchanged
        echo "$input"
        exit 0
        ;;
esac

# Extract failures from output
FAILURES=$(echo "$OUTPUT" | grep -E "^(FAIL|---\s*FAIL|✗|ERROR|undefined:|error:|panic:)" | head -20)
SUMMARY=$(echo "$OUTPUT" | grep -E "^(fmt:|lint:|test:|pint:|stan:|=== RESULT ===)" | tail -5)

# Also grab specific error lines with file:line references
FILE_ERRORS=$(echo "$OUTPUT" | grep -E "^[a-zA-Z0-9_/.-]+\.(go|php):[0-9]+:" | head -10)

if [ -z "$FAILURES" ] && [ "$EXIT_CODE" = "0" ]; then
    # All passed - show brief confirmation
    cat << 'EOF'
{
  "suppressOutput": true,
  "hookSpecificOutput": {
    "hookEventName": "PostToolUse",
    "additionalContext": "✓ QA passed"
  }
}
EOF
else
    # Combine failures and file errors
    ISSUES="$FAILURES"
    if [ -n "$FILE_ERRORS" ]; then
        ISSUES="$ISSUES
$FILE_ERRORS"
    fi

    # Escape for JSON
    ISSUES_ESCAPED=$(echo "$ISSUES" | sed 's/\\/\\\\/g' | sed 's/"/\\"/g' | sed ':a;N;$!ba;s/\n/\\n/g')
    SUMMARY_ESCAPED=$(echo "$SUMMARY" | sed 's/\\/\\\\/g' | sed 's/"/\\"/g' | sed ':a;N;$!ba;s/\n/ | /g')

    cat << EOF
{
  "suppressOutput": true,
  "hookSpecificOutput": {
    "hookEventName": "PostToolUse",
    "additionalContext": "## QA Issues\n\n\`\`\`\n$ISSUES_ESCAPED\n\`\`\`\n\n**Summary:** $SUMMARY_ESCAPED"
  }
}
EOF
fi
