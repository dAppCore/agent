#!/bin/bash
# Stop hook: Save session context to OpenBrain after each response
# Runs async so it doesn't block the conversation
#
# Debounced: only saves once per 10 minutes per project
# Only saves if there are git changes (real work happened)

BRAIN_URL="${CORE_BRAIN_URL:-https://api.lthn.sh}"
BRAIN_KEY="${CORE_BRAIN_KEY:-}"
BRAIN_KEY_FILE="${HOME}/.claude/brain.key"
DEBOUNCE_DIR="${HOME}/.claude/sessions/debounce"
DEBOUNCE_SECONDS=600  # 10 minutes

# Load API key
if [[ -z "$BRAIN_KEY" && -f "$BRAIN_KEY_FILE" ]]; then
    BRAIN_KEY=$(cat "$BRAIN_KEY_FILE" 2>/dev/null | tr -d '[:space:]')
fi

[[ -z "$BRAIN_KEY" ]] && exit 0

# Only save if there are uncommitted changes (real work happened)
GIT_STATUS=$(git status --short 2>/dev/null | head -5)
[[ -z "$GIT_STATUS" ]] && exit 0

# Detect project
PROJECT=""
CWD=$(pwd)
case "$CWD" in
    */core/go-*)   PROJECT=$(basename "$CWD" | sed 's/^go-//') ;;
    */core/php-*)  PROJECT=$(basename "$CWD" | sed 's/^php-//') ;;
    */core/*)      PROJECT=$(basename "$CWD") ;;
    */host-uk/*)   PROJECT=$(basename "$CWD") ;;
    */lthn/*)      PROJECT=$(basename "$CWD") ;;
    */snider/*)    PROJECT=$(basename "$CWD") ;;
esac
[[ -z "$PROJECT" ]] && PROJECT="unknown"

# Debounce: skip if we saved this project less than 10 min ago
mkdir -p "$DEBOUNCE_DIR"
DEBOUNCE_FILE="${DEBOUNCE_DIR}/${PROJECT}"
NOW=$(date '+%s')

if [[ -f "$DEBOUNCE_FILE" ]]; then
    LAST_SAVE=$(cat "$DEBOUNCE_FILE" 2>/dev/null)
    if [[ -n "$LAST_SAVE" ]]; then
        AGE=$((NOW - LAST_SAVE))
        [[ $AGE -lt $DEBOUNCE_SECONDS ]] && exit 0
    fi
fi

BRANCH=$(git branch --show-current 2>/dev/null || echo "none")
CHANGED_FILES=$(git diff --name-only 2>/dev/null | head -10 | tr '\n' ', ')
RECENT_COMMITS=$(git log --oneline -3 2>/dev/null | tr '\n' '; ')

CONTENT="Session on ${PROJECT} (branch: ${BRANCH}). Changed: ${CHANGED_FILES}Recent commits: ${RECENT_COMMITS}"

curl -s --max-time 5 "${BRAIN_URL}/v1/brain/remember" \
    -X POST \
    -H 'Content-Type: application/json' \
    -H 'Accept: application/json' \
    -H "Authorization: Bearer ${BRAIN_KEY}" \
    -d "{
        \"content\": $(echo "$CONTENT" | python3 -c 'import json,sys; print(json.dumps(sys.stdin.read().strip()))'),
        \"type\": \"context\",
        \"project\": \"${PROJECT}\",
        \"agent_id\": \"cladius\",
        \"tags\": [\"auto-save\", \"session\"]
    }" >/dev/null 2>&1

# Update debounce timestamp
echo "$NOW" > "$DEBOUNCE_FILE"

exit 0
