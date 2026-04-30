#!/bin/bash
# Check status of all agent workspaces.
# Usage: workspace-status.sh [repo-filter]

WS_ROOT="${CODE_PATH:-$HOME/Code}/.core/workspace"
FILTER="${1:-}"

if [ ! -d "$WS_ROOT" ]; then
  echo "No workspaces found"
  exit 0
fi

for ws in "$WS_ROOT"/*; do
  [ -d "$ws" ] || continue

  NAME=$(basename "$ws")

  # Apply filter
  if [ -n "$FILTER" ] && [[ "$NAME" != *"$FILTER"* ]]; then
    continue
  fi

  # Check for agent log
  LOG=$(ls "$ws"/agent-*.log 2>/dev/null | head -1)
  AGENT=$(basename "$LOG" .log 2>/dev/null | sed 's/agent-//')

  # Check PID if running
  PID_FILE="$ws/.pid"
  STATUS="completed"
  if [ -f "$PID_FILE" ]; then
    PID=$(cat "$PID_FILE")
    if ps -p "$PID" > /dev/null 2>&1; then
      STATUS="running (PID $PID)"
    fi
  fi

  # Check for commits
  COMMITS=0
  if [ -d "$ws/src/.git" ]; then
    COMMITS=$(cd "$ws/src" && git rev-list --count HEAD 2>/dev/null || echo 0)
  fi

  # Log size
  LOG_SIZE=0
  if [ -f "$LOG" ]; then
    LOG_SIZE=$(wc -c < "$LOG" | tr -d ' ')
  fi

  echo "$NAME | $AGENT | $STATUS | ${LOG_SIZE}b log | $COMMITS commits"
done
