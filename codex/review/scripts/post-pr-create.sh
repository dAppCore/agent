#!/bin/bash
# Suggest review after PR creation

read -r input
OUTPUT=$(echo "$input" | jq -r '.tool_response.stdout // .tool_response.output // empty')

# Extract PR URL from output
PR_URL=$(echo "$OUTPUT" | grep -oE 'https://github.com/[^/]+/[^/]+/pull/[0-9]+' | head -1)

if [ -n "$PR_URL" ]; then
    PR_NUM=$(echo "$PR_URL" | grep -oE '[0-9]+$')
    cat << EOF
{
  "hookSpecificOutput": {
    "hookEventName": "PostToolUse",
    "additionalContext": "PR created: $PR_URL\n\nRun \`/review:pr $PR_NUM\` to review before requesting reviewers."
  }
}
EOF
else
    echo "$input"
fi
