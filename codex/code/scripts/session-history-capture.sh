#!/bin/bash
# capture-session-history.sh
# Captures session context, focusing on git status, and saves it to history.json.

HISTORY_FILE="${HOME}/.claude/sessions/history.json"
SESSION_TIMEOUT=10800 # 3 hours

# Ensure session directory exists
mkdir -p "${HOME}/.claude/sessions"

# Initialize history file if it doesn't exist
if [[ ! -f "$HISTORY_FILE" ]]; then
    echo '{"sessions": []}' > "$HISTORY_FILE"
fi

# --- Get Session Identifiers ---
MODULE=$(basename "$(pwd)")
BRANCH=$(git rev-parse --abbrev-ref HEAD 2>/dev/null || echo "unknown")
NOW=$(date '+%s')

# --- Read and Find Current Session ---
HISTORY_CONTENT=$(cat "$HISTORY_FILE")
SESSION_INDEX=$(echo "$HISTORY_CONTENT" | jq \
    --arg module "$MODULE" \
    --arg branch "$BRANCH" \
    --argjson now "$NOW" \
    --argjson timeout "$SESSION_TIMEOUT" '
    .sessions | to_entries |
    map(select(.value.module == $module and .value.branch == $branch and ($now - .value.last_updated < $timeout))) |
    .[-1].key
')

# --- Extract Key Actions from Git ---
# Get list of modified/new files. `git status --short` gives entries like " M path/file.txt".
# We'll format them into more readable strings.
ACTIONS_LIST=()
while read -r line; do
    status=$(echo "$line" | cut -c 1-2)
    path=$(echo "$line" | cut -c 4-)
    action=""
    case "$status" in
        " M") action="Modified: $path" ;;
        "A ") action="Added: $path" ;;
        "D ") action="Deleted: $path" ;;
        "R ") action="Renamed: $path" ;;
        "C ") action="Copied: $path" ;;
        "??") action="Untracked: $path" ;;
    esac
    if [[ -n "$action" ]]; then
        ACTIONS_LIST+=("$action")
    fi
done < <(git status --short)

KEY_ACTIONS_JSON=$(printf '%s\n' "${ACTIONS_LIST[@]}" | jq -R . | jq -s .)

# --- Update or Create Session ---
if [[ "$SESSION_INDEX" != "null" ]]; then
    # Update existing session
    UPDATED_HISTORY=$(echo "$HISTORY_CONTENT" | jq \
        --argjson index "$SESSION_INDEX" \
        --argjson ts "$NOW" \
        --argjson actions "$KEY_ACTIONS_JSON" '
        .sessions[$index].last_updated = $ts |
        .sessions[$index].key_actions = $actions
        # Note: pending_tasks and decisions would be updated here from conversation
        '
    )
else
    # Create new session
    SESSION_ID="session_$(date '+%Y%m%d%H%M%S')_$$"
    NEW_SESSION=$(jq -n \
        --arg id "$SESSION_ID" \
        --argjson ts "$NOW" \
        --arg module "$MODULE" \
        --arg branch "$BRANCH" \
        --argjson actions "$KEY_ACTIONS_JSON" '
        {
            "id": $id,
            "started": $ts,
            "last_updated": $ts,
            "module": $module,
            "branch": $branch,
            "key_actions": $actions,
            "pending_tasks": [],
            "decisions": []
        }'
    )
    UPDATED_HISTORY=$(echo "$HISTORY_CONTENT" | jq --argjson new_session "$NEW_SESSION" '.sessions += [$new_session]')
fi

# Write back to file
# Use a temp file for atomic write
TMP_FILE="${HISTORY_FILE}.tmp"
echo "$UPDATED_HISTORY" > "$TMP_FILE" && mv "$TMP_FILE" "$HISTORY_FILE"

# This script does not produce output, it works in the background.
exit 0
