#!/bin/bash
# Remind about verification before push

read -r input

# Check if tests were run recently (within last 5 minutes)
LAST_TEST=$(find . -name "*.test" -mmin -5 2>/dev/null | head -1)
LAST_COVERAGE=$(find . -name "coverage.*" -mmin -5 2>/dev/null | head -1)

if [ -z "$LAST_TEST" ] && [ -z "$LAST_COVERAGE" ]; then
    cat << 'EOF'
{
  "hookSpecificOutput": {
    "hookEventName": "PreToolUse",
    "additionalContext": "⚠️ No recent test run detected. Consider running `/core:verify` before pushing."
  }
}
EOF
else
    echo "$input"
fi
