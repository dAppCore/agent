#!/bin/bash
# Post-commit hook: Check for uncommitted work that might get lost
# Policy: EXPOSE warning when uncommitted work exists, HIDE when clean

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/output-policy.sh"

read -r input
COMMAND=$(echo "$input" | jq -r '.tool_input.command // empty')

# Only run after git commit
if ! echo "$COMMAND" | grep -qE '^git commit'; then
    pass_through "$input"
    exit 0
fi

# Check for remaining uncommitted changes
UNSTAGED=$(git diff --name-only 2>/dev/null | wc -l | tr -d ' ')
STAGED=$(git diff --cached --name-only 2>/dev/null | wc -l | tr -d ' ')
UNTRACKED=$(git ls-files --others --exclude-standard 2>/dev/null | wc -l | tr -d ' ')

TOTAL=$((UNSTAGED + STAGED + UNTRACKED))

if [[ $TOTAL -gt 0 ]]; then
    DETAILS=""

    if [[ $UNSTAGED -gt 0 ]]; then
        FILES=$(git diff --name-only 2>/dev/null | head -5 | sed 's/^/  - /')
        DETAILS+="**Modified (unstaged):** $UNSTAGED files\n$FILES\n"
        [[ $UNSTAGED -gt 5 ]] && DETAILS+="  ... and $((UNSTAGED - 5)) more\n"
    fi

    if [[ $STAGED -gt 0 ]]; then
        FILES=$(git diff --cached --name-only 2>/dev/null | head -5 | sed 's/^/  - /')
        DETAILS+="**Staged (not committed):** $STAGED files\n$FILES\n"
    fi

    if [[ $UNTRACKED -gt 0 ]]; then
        FILES=$(git ls-files --others --exclude-standard 2>/dev/null | head -5 | sed 's/^/  - /')
        DETAILS+="**Untracked:** $UNTRACKED files\n$FILES\n"
        [[ $UNTRACKED -gt 5 ]] && DETAILS+="  ... and $((UNTRACKED - 5)) more\n"
    fi

    expose_warning "Uncommitted work remains ($TOTAL files)" "$DETAILS"
else
    pass_through "$input"
fi
