#!/bin/bash
# Show CI status hint after push

read -r input
EXIT_CODE=$(echo "$input" | jq -r '.tool_response.exit_code // 0')

if [ "$EXIT_CODE" = "0" ]; then
    # Check if repo has workflows
    if [ -d ".github/workflows" ]; then
        cat << 'EOF'
{
  "hookSpecificOutput": {
    "hookEventName": "PostToolUse",
    "additionalContext": "Push successful. CI workflows will run shortly.\n\nRun `/ci:status` to check progress or `gh run watch` to follow live."
  }
}
EOF
    else
        echo "$input"
    fi
else
    echo "$input"
fi
