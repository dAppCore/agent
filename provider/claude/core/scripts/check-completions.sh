#!/bin/bash
# Check for agent completion events since last check.
# Called by plugin hooks to notify the orchestrating agent.

EVENTS_FILE="${CORE_WORKSPACE:-$HOME/Code/.core}/workspace/events.jsonl"
MARKER_FILE="${CORE_WORKSPACE:-$HOME/Code/.core}/workspace/.events-read"

if [ ! -f "$EVENTS_FILE" ]; then
    exit 0
fi

# Get events newer than last read
if [ -f "$MARKER_FILE" ]; then
    LAST_READ=$(cat "$MARKER_FILE")
    NEW_EVENTS=$(awk -v ts="$LAST_READ" '$0 ~ "timestamp" && $0 > ts' "$EVENTS_FILE" 2>/dev/null)
else
    NEW_EVENTS=$(cat "$EVENTS_FILE")
fi

if [ -z "$NEW_EVENTS" ]; then
    exit 0
fi

# Update marker
date -u +%Y-%m-%dT%H:%M:%SZ > "$MARKER_FILE"

# Count completions
COUNT=$(echo "$NEW_EVENTS" | grep -c "agent_completed")

if [ "$COUNT" -gt 0 ]; then
    # Format for hook output
    AGENTS=$(echo "$NEW_EVENTS" | grep "agent_completed" | python3 -c "
import sys, json
events = [json.loads(l) for l in sys.stdin if l.strip()]
for e in events:
    print(f\"  {e.get('agent','?')} — {e.get('workspace','?')}\")
" 2>/dev/null)

    cat << EOF
{
  "hookSpecificOutput": {
    "hookEventName": "Notification",
    "additionalContext": "$COUNT agent(s) completed:\n$AGENTS\n\nRun /core:status to review."
  }
}
EOF
fi
