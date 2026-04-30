#!/bin/bash
# Pre-compact: Save state locally + to OpenBrain before context compaction
#
# Captures:
# - Working directory + branch
# - Git status (files touched)
# - Todo state (in_progress items)
# - Saves to OpenBrain so the memory survives across sessions

STATE_FILE="${HOME}/.claude/sessions/scratchpad.md"
BRAIN_URL="${CORE_BRAIN_URL:-https://api.lthn.sh}"
BRAIN_KEY="${CORE_BRAIN_KEY:-}"
BRAIN_KEY_FILE="${HOME}/.claude/brain.key"
TIMESTAMP=$(date '+%s')
CWD=$(pwd)

mkdir -p "${HOME}/.claude/sessions"

# Load API key
if [[ -z "$BRAIN_KEY" && -f "$BRAIN_KEY_FILE" ]]; then
    BRAIN_KEY=$(cat "$BRAIN_KEY_FILE" 2>/dev/null | tr -d '[:space:]')
fi

# Get git status
GIT_STATUS=""
BRANCH=""
if git rev-parse --git-dir > /dev/null 2>&1; then
    GIT_STATUS=$(git status --short 2>/dev/null | head -15)
    BRANCH=$(git branch --show-current 2>/dev/null)
fi

# Detect project
PROJECT=""
case "$CWD" in
    */core/go-*)   PROJECT=$(basename "$CWD" | sed 's/^go-//') ;;
    */core/php-*)  PROJECT=$(basename "$CWD" | sed 's/^php-//') ;;
    */core/*)      PROJECT=$(basename "$CWD") ;;
    */host-uk/*)   PROJECT=$(basename "$CWD") ;;
    */lthn/*)      PROJECT=$(basename "$CWD") ;;
    */snider/*)    PROJECT=$(basename "$CWD") ;;
esac

# Save local scratchpad
cat > "$STATE_FILE" << EOF
---
timestamp: ${TIMESTAMP}
cwd: ${CWD}
branch: ${BRANCH:-none}
---

# Resume After Compact

You were mid-task. Do NOT assume work is complete.

## Project
\`${CWD}\` on \`${BRANCH:-no branch}\`

## Files Changed
\`\`\`
${GIT_STATUS:-none}
\`\`\`

## Next
Continue the in_progress todo. Check OpenBrain for context.
EOF

echo "[PreCompact] Local snapshot saved" >&2

# Save to OpenBrain
if [[ -n "$BRAIN_KEY" && -n "$GIT_STATUS" ]]; then
    CHANGED_FILES=$(echo "$GIT_STATUS" | awk '{print $2}' | tr '\n' ', ')
    CONTENT="Context compaction during work on ${PROJECT:-unknown} (branch: ${BRANCH:-none}). Changed files: ${CHANGED_FILES}"

    curl -s --max-time 5 "${BRAIN_URL}/v1/brain/remember" \
        -X POST \
        -H 'Content-Type: application/json' \
        -H 'Accept: application/json' \
        -H "Authorization: Bearer ${BRAIN_KEY}" \
        -d "{
            \"content\": $(echo "$CONTENT" | python3 -c 'import json,sys; print(json.dumps(sys.stdin.read().strip()))'),
            \"type\": \"context\",
            \"project\": \"${PROJECT:-unknown}\",
            \"agent_id\": \"cladius\",
            \"tags\": [\"pre-compact\", \"session\"]
        }" >/dev/null 2>&1

    echo "[PreCompact] Saved to OpenBrain" >&2
fi

exit 0
