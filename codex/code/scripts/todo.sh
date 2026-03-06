#!/bin/bash
set -euo pipefail
IFS=$'\n\t'

# Function to process and format TODOs
process_todos() {
  local sort_by_priority=false
  if [[ "${1:-}" == "--priority" ]]; then
    sort_by_priority=true
  fi

  local count=0
  local high=0
  local med=0
  local low=0
  local output=""
  local found_todos=false

  while IFS= read -r line; do
    found_todos=true
    ((count++))
    filepath=$(echo "$line" | cut -d: -f1)
    linenumber=$(echo "$line" | cut -d: -f2)

    message_raw=$(echo "$line" | cut -d: -f3-)
    message=$(echo "$message_raw" | sed -e 's/^[[:space:]]*\/\///' -e 's/^[[:space:]]*#//' -e 's/^[[:space:]]*//' | sed -e 's/TODO:|FIXME:|HACK:|XXX://i' | sed 's/^[[:space:]]*//')

    sort_key=2
    priority="MED"
    if [[ $line =~ FIXME: || ($line =~ TODO: && $line =~ urgent) ]]; then
      priority="HIGH"
      sort_key=1
      ((high++))
    elif [[ $line =~ HACK: || $line =~ XXX: ]]; then
      priority="LOW"
      sort_key=3
      ((low++))
    else
      ((med++))
    fi

    if git ls-files --error-unmatch "$filepath" >/dev/null 2>&1; then
      age=$(git log -1 --format=%ar -- "$filepath")
    else
      age="untracked"
    fi

    formatted_line=$(printf "%d_#%s [%s] %s\n   %s:%s\n   Added: %s\n\n" "$sort_key" "$count" "$priority" "$message" "$filepath" "$linenumber" "$age")
    output+="$formatted_line"
  done < <(grep -r -n -i -E "TODO:|FIXME:|HACK:|XXX:" . \
    --exclude-dir=".git" \
    --exclude-dir=".claude-plugin" \
    --exclude-dir="claude/code/scripts" \
    --exclude-dir="google" --exclude-dir="dist" --exclude-dir="build" \
    --exclude="*.log" --exclude="todos.txt" --exclude="test_loop.sh" || true)

  if [ "$found_todos" = false ]; then
    echo "No TODOs found."
  else
    if [[ "$sort_by_priority" = true ]]; then
      echo -e "$output" | sort -n | sed 's/^[0-9]_//'
    else
      echo -e "$output" | sed 's/^[0-9]_//'
    fi
    echo "Total: $count TODOs ($high high, $med medium, $low low)"
  fi
}

# Default action is to list TODOs
ACTION="list"
ARGS=""

# Parse command-line arguments
if [[ $# -gt 0 ]]; then
  if [[ "$1" == "--priority" ]]; then
    ACTION="--priority"
    shift
  else
    ACTION="$1"
    shift
  fi
  ARGS="$@"
fi

case "$ACTION" in
  list)
    process_todos
    ;;
  add)
    echo "Error: 'add' command not implemented." >&2
    exit 1
    ;;
  done)
    echo "Error: 'done' command not implemented." >&2
    exit 1
    ;;
  --priority)
    process_todos --priority
    ;;
  *)
    echo "Usage: /core:todo [list | --priority]" >&2
    exit 1
    ;;
esac
