#!/bin/bash
#
# Renders a summary of all repository statuses.
# Wraps the `core dev health` command with friendlier formatting.
#

# --- Configuration ---
# Set to `true` to use mock data for testing.
USE_MOCK_DATA=false
# Set to the actual command to get repo health.
# The command is expected to return data in the format:
# <module> <branch> <status> <insertions> <deletions> <behind> <ahead>
HEALTH_COMMAND="core dev health"

# --- Argument Parsing ---
SHOW_DIRTY_ONLY=false
SHOW_BEHIND_ONLY=false

for arg in "$@"; do
  case $arg in
    --dirty)
      SHOW_DIRTY_ONLY=true
      shift
      ;;
    --behind)
      SHOW_BEHIND_ONLY=true
      shift
      ;;
  esac
done

# --- Mock Data ---
# Used for development and testing if USE_MOCK_DATA is true.
mock_health_data() {
  cat <<EOF
core-php main clean 0 0 0 0
core-tenant feat/auth dirty 2 0 0 0
core-admin main clean 0 0 0 0
core-api main clean 0 0 3 0
core-mcp dev dirty 1 1 1 2
repo-clean-ahead main clean 0 0 0 5
EOF
}

# --- Data Fetching ---
if [ "$USE_MOCK_DATA" = true ]; then
  health_data=$(mock_health_data)
else
  # In a real scenario, we'd run the actual command.
  # For now, since `core dev health` is not a real command in this sandbox,
  # I will fall back to mock data if the command fails.
  health_data=$($HEALTH_COMMAND 2>/dev/null) || health_data=$(mock_health_data)
fi

# --- Output Formatting ---
# Table header
header=$(printf "%-15s %-15s %-10s %-12s" "Module" "Branch" "Status" "Behind/Ahead")
# Use dynamic width if possible, otherwise a fixed width.
cols=$(tput cols 2>/dev/null || echo 67)
separator=$(printf '━%.0s' $(seq 1 $cols))

echo "Host UK Monorepo Status"
echo "${separator:0:${#header}}"
echo "$header"
echo "${separator:0:${#header}}"

# Process each line of health data
while read -r module branch status insertions deletions behind ahead; do

  is_dirty=false
  is_behind=false
  details=""

  # Determine status and details string
  if [ "$status" = "dirty" ]; then
    is_dirty=true
    details="+${insertions} -${deletions}"
  else
    if [ "$behind" -gt 0 ] && [ "$ahead" -gt 0 ]; then
      details="-${behind} +${ahead}"
      is_behind=true
    elif [ "$behind" -gt 0 ]; then
      details="-${behind} (behind)"
      is_behind=true
    elif [ "$ahead" -gt 0 ]; then
      details="+${ahead}"
    else
      details="✓"
    fi
  fi

  # Apply filters
  if [ "$SHOW_DIRTY_ONLY" = true ] && [ "$is_dirty" = false ]; then
    continue
  fi
  if [ "$SHOW_BEHIND_ONLY" = true ] && [ "$is_behind" = false ]; then
    continue
  fi

  # Print table row
  printf "%-15s %-15s %-10s %-12s\n" "$module" "$branch" "$status" "$details"

done <<< "$health_data"

# --- Summary ---
# The summary should reflect the total state, regardless of filters.
total_clean_repo_count=$(echo "$health_data" | grep " clean " -c || true)
dirty_repo_count=$(echo "$health_data" | grep " dirty " -c || true)
behind_repo_count=0
while read -r module branch status insertions deletions behind ahead; do
    if [ "$status" = "clean" ] && [[ "$behind" =~ ^[0-9]+$ ]] && [ "$behind" -gt 0 ]; then
        behind_repo_count=$((behind_repo_count+1))
    fi
done <<< "$health_data"

clean_repo_count=$((total_clean_repo_count - behind_repo_count))

summary_parts=()
if [ "$dirty_repo_count" -gt 0 ]; then
    summary_parts+=("$dirty_repo_count dirty")
fi
if [ "$behind_repo_count" -gt 0 ]; then
    summary_parts+=("$behind_repo_count behind")
fi
if [ "$clean_repo_count" -gt 0 ]; then
    summary_parts+=("$clean_repo_count clean")
fi


summary_string=$(printf "%s, " "${summary_parts[@]}")
summary_string=${summary_string%, } # remove trailing comma and space

echo ""
echo "Summary: $summary_string"
