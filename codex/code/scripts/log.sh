#!/bin/bash

# Smart log viewing for laravel.log

LOG_FILE="storage/logs/laravel.log"

# Check if log file exists
if [ ! -f "$LOG_FILE" ]; then
  echo "Error: Log file not found at $LOG_FILE"
  exit 1
fi

# --- Argument Parsing ---

# Default action: tail log file
if [ -z "$1" ]; then
  tail -f "$LOG_FILE"
  exit 0
fi

case "$1" in
  --errors)
    grep "\.ERROR" "$LOG_FILE"
    ;;

  --since)
    if [ -z "$2" ]; then
      echo "Error: Missing duration for --since (e.g., 1h, 30m, 2d)"
      exit 1
    fi
    # Simple parsing for duration
    duration_string=$(echo "$2" | sed 's/h/ hours/' | sed 's/m/ minutes/' | sed 's/d/ days/')
    since_date=$(date -d "now - $duration_string" '+%Y-%m-%d %H:%M:%S' 2>/dev/null)

    if [ -z "$since_date" ]; then
        echo "Error: Invalid duration format. Use formats like '1h', '30m', '2d'."
        exit 1
    fi

    awk -v since="$since_date" '
      {
        # Extract timestamp like "2024-01-15 10:30:45" from "[2024-01-15 10:30:45]"
        log_ts = substr($1, 2) " " substr($2, 1, 8)
        if (log_ts >= since) {
            print $0
        }
      }
    ' "$LOG_FILE"
    ;;

  --grep)
    if [ -z "$2" ]; then
      echo "Error: Missing pattern for --grep"
      exit 1
    fi
    grep -E "$2" "$LOG_FILE"
    ;;

  --request)
    if [ -z "$2" ]; then
      echo "Error: Missing request ID for --request"
      exit 1
    fi
    grep "\"request_id\":\"$2\"" "$LOG_FILE"
    ;;

  analyse)
    echo "Log Analysis: Last 24 hours"
    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    echo ""

    since_date_24h=$(date -d "now - 24 hours" '+%Y-%m-%d %H:%M:%S')

    log_entries_24h=$(awk -v since="$since_date_24h" '
      {
        log_ts = substr($1, 2) " " substr($2, 1, 8)
        if (log_ts >= since) {
            print $0
        }
      }
    ' "$LOG_FILE")

    if [ -z "$log_entries_24h" ]; then
        echo "No log entries in the last 24 hours."
        exit 0
    fi

    total_entries=$(echo "$log_entries_24h" | wc -l)
    error_entries=$(echo "$log_entries_24h" | grep -c "\.ERROR" || true)
    warning_entries=$(echo "$log_entries_24h" | grep -c "\.WARNING" || true)
    info_entries=$(echo "$log_entries_24h" | grep -c "\.INFO" || true)

    echo "Total entries: $total_entries"
    echo "Errors: $error_entries"
    echo "Warnings: $warning_entries"
    echo "Info: $info_entries"
    echo ""

    if [ "$error_entries" -gt 0 ]; then
      echo "Top Errors:"

      error_lines=$(echo "$log_entries_24h" | grep "\.ERROR")

      top_errors=$(echo "$error_lines" | \
        sed -E 's/.*\.([A-Z]+): //' | \
        sed 's/ in .*//' | \
        sort | uniq -c | sort -nr | head -n 3)

      i=1
      echo "$top_errors" | while read -r line; do
        count=$(echo "$line" | awk '{print $1}')
        error_name=$(echo "$line" | awk '{$1=""; print $0}' | sed 's/^ //')

        # Find a representative location
        location=$(echo "$error_lines" | grep -m 1 "$error_name" | grep " in " | sed 's/.* in //')

        echo "$i. $error_name ($count times)"
        if [ ! -z "$location" ]; then
          echo "   $location"
        else
          # For cases like ValidationException
          if echo "$error_name" | grep -q "ValidationException"; then
             echo "   Various controllers"
          fi
        fi
        echo ""
        i=$((i+1))
      done

      if echo "$top_errors" | grep -q "TokenExpiredException"; then
        echo "Recommendations:"
        echo "- TokenExpiredException happening frequently"
        echo "  Consider increasing token lifetime or"
        echo "  implementing automatic refresh"
        echo ""
      fi
    fi
    ;;

  *)
    echo "Invalid command: $1"
    echo "Usage: /core:log [--errors|--since <duration>|--grep <pattern>|--request <id>|analyse]"
    exit 1
    ;;
esac
