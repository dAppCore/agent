#!/bin/bash

# Fetch the raw status from the core dev health command.
# The output format is assumed to be:
# module branch status ahead behind insertions deletions
RAW_STATUS=$(core dev health 2>/dev/null)

# Exit if the command fails or produces no output
if [ -z "$RAW_STATUS" ]; then
  echo "Failed to get repo status from 'core dev health'."
  echo "Make sure the 'core' command is available and repositories are correctly configured."
  exit 1
fi

FILTER="$1"

# --- Header ---
echo "Host UK Monorepo Status"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
printf "%-15s %-15s %-10s %s\n" "Module" "Branch" "Status" "Behind/Ahead"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"

# --- Data Processing and Printing ---
while read -r module branch status ahead behind insertions deletions; do
    is_dirty=false
    is_behind=false

    if [[ "$status" == "dirty" ]]; then
        is_dirty=true
    fi

    if (( behind > 0 )); then
        is_behind=true
    fi

    # Apply filters
    if [[ "$FILTER" == "--dirty" && "$is_dirty" == "false" ]]; then
        continue
    fi
    if [[ "$FILTER" == "--behind" && "$is_behind" == "false" ]]; then
        continue
    fi

    # Format the "Behind/Ahead" column based on status
    if [[ "$status" == "dirty" ]]; then
        behind_ahead_text="+${insertions} -${deletions}"
    else # status is 'clean'
        if (( behind > 0 )); then
            behind_ahead_text="-${behind} (behind)"
        elif (( ahead > 0 )); then
             behind_ahead_text="+${ahead}"
        else
            behind_ahead_text="✓"
        fi
    fi

    printf "%-15s %-15s %-10s %s\n" "$module" "$branch" "$status" "$behind_ahead_text"

done <<< "$RAW_STATUS"

# --- Summary ---
# The summary is always based on the full, unfiltered data.
dirty_count=$(echo "$RAW_STATUS" | grep -cw "dirty")
behind_count=$(echo "$RAW_STATUS" | awk '($5+0) > 0' | wc -l)
clean_count=$(echo "$RAW_STATUS" | grep -cw "clean")

summary_parts=()
if (( dirty_count > 0 )); then
    summary_parts+=("$dirty_count dirty")
fi
if (( behind_count > 0 )); then
    summary_parts+=("$behind_count behind")
fi
summary_parts+=("$clean_count clean")

summary="Summary: $(IFS=, ; echo "${summary_parts[*]}")"

echo
echo "$summary"
