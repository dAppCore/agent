#!/bin/bash
# Ensure work ends with a commit during /core:yes mode
#
# Stop hook that blocks if uncommitted changes exist.
# Prevents Claude from stopping before work is committed.

read -r input
STOP_ACTIVE=$(echo "$input" | jq -r '.stop_hook_active // false')

# Prevent infinite loop - if we already blocked once, allow stop
if [ "$STOP_ACTIVE" = "true" ]; then
    exit 0
fi

# Check for uncommitted changes
UNSTAGED=$(git diff --name-only 2>/dev/null | wc -l | tr -d ' ')
STAGED=$(git diff --cached --name-only 2>/dev/null | wc -l | tr -d ' ')
UNTRACKED=$(git ls-files --others --exclude-standard 2>/dev/null | grep -v '^\.idea/' | wc -l | tr -d ' ')

TOTAL=$((UNSTAGED + STAGED + UNTRACKED))

if [ "$TOTAL" -gt 0 ]; then
    # Build file list for context
    FILES=""
    if [ "$UNSTAGED" -gt 0 ]; then
        FILES="$FILES\nModified: $(git diff --name-only 2>/dev/null | head -3 | tr '\n' ' ')"
    fi
    if [ "$STAGED" -gt 0 ]; then
        FILES="$FILES\nStaged: $(git diff --cached --name-only 2>/dev/null | head -3 | tr '\n' ' ')"
    fi
    if [ "$UNTRACKED" -gt 0 ]; then
        FILES="$FILES\nUntracked: $(git ls-files --others --exclude-standard 2>/dev/null | grep -v '^\.idea/' | head -3 | tr '\n' ' ')"
    fi

    cat << EOF
{
  "decision": "block",
  "reason": "You have $TOTAL uncommitted changes. Please commit them before stopping.\n$FILES\n\nUse: git add <files> && git commit -m 'type(scope): description'"
}
EOF
else
    # No changes, allow stop
    exit 0
fi
