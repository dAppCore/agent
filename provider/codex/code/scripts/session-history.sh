#!/bin/bash
# Manage session history in ~/.claude/sessions/history.json

HISTORY_FILE="${HOME}/.claude/sessions/history.json"
SESSION_ID="${CLAUDE_SESSION_ID:-$(date +%s)-${RANDOM}}"
SEVEN_DAYS=604800 # seconds

# Ensure the sessions directory and history file exist
mkdir -p "${HOME}/.claude/sessions"
if [[ ! -f "$HISTORY_FILE" ]]; then
    echo '{"sessions": []}' > "$HISTORY_FILE"
fi

# Function to get the current session
get_session() {
    jq --arg id "$SESSION_ID" '.sessions[] | select(.id == $id)' "$HISTORY_FILE"
}

# Function to create or update the session
touch_session() {
    local module_name="$(basename "$PWD")"
    local branch_name="$(git branch --show-current 2>/dev/null || echo 'unknown')"

    if [[ -z "$(get_session)" ]]; then
        # Create new session
        jq --arg id "$SESSION_ID" --arg started "$(date -u +%Y-%m-%dT%H:%M:%SZ)" \
           --arg module "$module_name" --arg branch "$branch_name" \
           '.sessions += [{
               "id": $id,
               "started": $started,
               "module": $module,
               "branch": $branch,
               "key_actions": [],
               "pending_tasks": [],
               "decisions": []
           }]' "$HISTORY_FILE" > "${HISTORY_FILE}.tmp" && mv "${HISTORY_FILE}.tmp" "$HISTORY_FILE"
    fi
}

# Function to add an entry to a session array (key_actions, pending_tasks, decisions)
add_to_session() {
    local type="$1" # e.g., "key_actions"
    local content="$2"

    touch_session
    jq --arg id "$SESSION_ID" --arg type "$type" --arg content "$content" \
       '( .sessions[] | select(.id == $id) | .[$type] ) |= (. + [$content])' \
       "$HISTORY_FILE" > "${HISTORY_FILE}.tmp" && mv "${HISTORY_FILE}.tmp" "$HISTORY_FILE"
}

# Function to prune old sessions
prune_sessions() {
    local now
    now=$(date +%s)
    jq --argjson seven_days "$SEVEN_DAYS" --argjson now "$now" \
        '.sessions |= map(select( (($now - (.started | fromdate)) < $seven_days) ))' \
        "$HISTORY_FILE" > "${HISTORY_FILE}.tmp" && mv "${HISTORY_FILE}.tmp" "$HISTORY_FILE"
}

# --- Main script logic ---
COMMAND="$1"
shift

case "$COMMAND" in
    "start")
        touch_session
        prune_sessions
        ;;
    "action")
        add_to_session "key_actions" "$1"
        ;;
    "task")
        add_to_session "pending_tasks" "$1"
        ;;
    "decision")
        add_to_session "decisions" "$1"
        ;;
    "show")
        # Display the most recent session
        jq '.sessions | sort_by(.started) | .[-1]' "$HISTORY_FILE"
        ;;
    *)
        echo "Usage: $0 {start|action|task|decision|show} [content]" >&2
        exit 1
        ;;
esac

exit 0
