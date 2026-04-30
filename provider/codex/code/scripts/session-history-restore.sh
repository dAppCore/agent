#!/bin/bash
# session-history-restore.sh
# Restores and displays the most recent session context from history.json.

HISTORY_FILE="${HOME}/.claude/sessions/history.json"
PRUNE_AGE_DAYS=7 # Prune sessions older than 7 days

# Ensure the history file exists, otherwise exit silently.
if [[ ! -f "$HISTORY_FILE" ]]; then
    exit 0
fi

# --- Prune Old Sessions ---
NOW=$(date '+%s')
PRUNE_TIMESTAMP=$((NOW - PRUNE_AGE_DAYS * 86400))
PRUNED_HISTORY=$(jq --argjson prune_ts "$PRUNE_TIMESTAMP" '
    .sessions = (.sessions | map(select(.last_updated >= $prune_ts)))
' "$HISTORY_FILE")

# Atomically write the pruned history back to the file
TMP_FILE="${HISTORY_FILE}.tmp"
echo "$PRUNED_HISTORY" > "$TMP_FILE" && mv "$TMP_FILE" "$HISTORY_FILE"

# --- Read the Most Recent Session ---
# Get the last session from the (potentially pruned) history
LAST_SESSION=$(echo "$PRUNED_HISTORY" | jq '.sessions[-1]')

# If no sessions, exit.
if [[ "$LAST_SESSION" == "null" ]]; then
    exit 0
fi

# --- Format and Display Session Context ---
MODULE=$(echo "$LAST_SESSION" | jq -r '.module')
BRANCH=$(echo "$LAST_SESSION" | jq -r '.branch')
LAST_UPDATED=$(echo "$LAST_SESSION" | jq -r '.last_updated')

# Calculate human-readable "last active" time
AGE_SECONDS=$((NOW - LAST_UPDATED))
if (( AGE_SECONDS < 60 )); then
    LAST_ACTIVE="less than a minute ago"
elif (( AGE_SECONDS < 3600 )); then
    LAST_ACTIVE="$((AGE_SECONDS / 60)) minutes ago"
elif (( AGE_SECONDS < 86400 )); then
    LAST_ACTIVE="$((AGE_SECONDS / 3600)) hours ago"
else
    LAST_ACTIVE="$((AGE_SECONDS / 86400)) days ago"
fi

# --- Build the Output ---
# Using ANSI escape codes for formatting (bold, colors)
BLUE='\033[0;34m'
BOLD='\033[1m'
NC='\033[0m' # No Color

# Header
echo -e "${BLUE}${BOLD}📋 Previous Session Context${NC}" >&2
echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}" >&2
echo -e "${BOLD}Module:${NC} ${MODULE} (${BRANCH})" >&2
echo -e "${BOLD}Last active:${NC} ${LAST_ACTIVE}" >&2
echo "" >&2

# Key Actions
KEY_ACTIONS=$(echo "$LAST_SESSION" | jq -r '.key_actions[]?')
if [[ -n "$KEY_ACTIONS" ]]; then
    echo -e "${BOLD}Key actions:${NC}" >&2
    while read -r action; do
        echo -e "• ${action}" >&2
    done <<< "$KEY_ACTIONS"
    echo "" >&2
fi

# Pending Tasks
PENDING_TASKS=$(echo "$LAST_SESSION" | jq -r '.pending_tasks[]?')
if [[ -n "$PENDING_TASKS" ]]; then
    echo -e "${BOLD}Pending tasks:${NC}" >&2
    while read -r task; do
        echo -e "• ${task}" >&2
    done <<< "$PENDING_TASKS"
    echo "" >&2
fi

# Decisions Made
DECISIONS=$(echo "$LAST_SESSION" | jq -r '.decisions[]?')
if [[ -n "$DECISIONS" ]]; then
    echo -e "${BOLD}Decisions made:${NC}" >&2
    while read -r decision; do
        echo -e "• ${decision}" >&2
    done <<< "$DECISIONS"
    echo "" >&2
fi

exit 0
