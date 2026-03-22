#!/bin/bash
# Check for new inbox messages since last check.
# Silent if no new messages. Only outputs when there's something new.

MARKER_FILE="${CORE_WORKSPACE:-$HOME/Code/.core}/workspace/.inbox-last-id"
BRAIN_KEY=$(cat "$HOME/.claude/brain.key" 2>/dev/null | tr -d '\n')

if [ -z "$BRAIN_KEY" ]; then
    exit 0
fi

# Get agent name
HOSTNAME=$(hostname | tr '[:upper:]' '[:lower:]')
if echo "$HOSTNAME" | grep -qE "snider|studio|mac"; then
    AGENT="cladius"
else
    AGENT="charon"
fi

# Fetch inbox
RESPONSE=$(curl -sf -H "Authorization: Bearer $BRAIN_KEY" \
    "https://api.lthn.sh/v1/messages/inbox?agent=$AGENT" 2>/dev/null)

if [ -z "$RESPONSE" ]; then
    exit 0
fi

# Get last seen ID
LAST_ID=0
if [ -f "$MARKER_FILE" ]; then
    LAST_ID=$(cat "$MARKER_FILE" | tr -d '\n')
fi

# Check for new messages
NEW=$(echo "$RESPONSE" | python3 -c "
import sys, json
data = json.load(sys.stdin)
msgs = data.get('messages') or data.get('data') or []
last_id = int(${LAST_ID})
new_msgs = [m for m in msgs if m.get('id', 0) > last_id]
if not new_msgs:
    sys.exit(0)
# Update marker to highest ID
max_id = max(m['id'] for m in new_msgs)
print(f'NEW_ID={max_id}')
for m in new_msgs:
    print(f\"  {m.get('from_agent', m.get('from', '?'))}: {m.get('subject', '(no subject)')}\")
" 2>/dev/null)

if [ -z "$NEW" ]; then
    exit 0
fi

# Extract new max ID and update marker
NEW_ID=$(echo "$NEW" | grep "^NEW_ID=" | cut -d= -f2)
if [ -n "$NEW_ID" ]; then
    echo "$NEW_ID" > "$MARKER_FILE"
fi

# Output for Claude
DETAILS=$(echo "$NEW" | grep -v "^NEW_ID=")
COUNT=$(echo "$DETAILS" | wc -l | tr -d ' ')

echo "New inbox: $COUNT message(s)"
echo "$DETAILS"
echo "Use agent_inbox to read full messages."
