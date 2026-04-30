#!/bin/bash
# Calculate and display test coverage.

set -e

COVERAGE_HISTORY_FILE=".coverage-history.json"

# --- Helper Functions ---

# TODO: Replace this with the actual command to calculate test coverage
get_current_coverage() {
    echo "80.0" # Mock value
}

get_previous_coverage() {
    if [ ! -f "$COVERAGE_HISTORY_FILE" ] || ! jq -e '.history | length > 0' "$COVERAGE_HISTORY_FILE" > /dev/null 2>&1; then
        echo "0.0"
        return
    fi
    jq -r '.history[-1].coverage' "$COVERAGE_HISTORY_FILE"
}

update_history() {
    local coverage=$1
    local commit_hash=$(git rev-parse HEAD)
    local timestamp=$(date -u +"%Y-%m-%dT%H:%M:%SZ")

    if [ ! -f "$COVERAGE_HISTORY_FILE" ]; then
        echo '{"history": []}' > "$COVERAGE_HISTORY_FILE"
    fi

    local updated_history=$(jq \
        --arg commit "$commit_hash" \
        --arg date "$timestamp" \
        --argjson coverage "$coverage" \
        '.history += [{ "commit": $commit, "date": $date, "coverage": $coverage }]' \
        "$COVERAGE_HISTORY_FILE")

    echo "$updated_history" > "$COVERAGE_HISTORY_FILE"
}

# --- Main Logic ---

handle_diff() {
    local current_coverage=$(get_current_coverage)
    local previous_coverage=$(get_previous_coverage)
    local change=$(awk -v current="$current_coverage" -v previous="$previous_coverage" 'BEGIN {printf "%.2f", current - previous}')

    echo "Test Coverage Report"
    echo "━━━━━━━━━━━━━━━━━━━━"
    echo "Current:  $current_coverage%"
    echo "Previous: $previous_coverage%"

    if awk -v change="$change" 'BEGIN {exit !(change >= 0)}'; then
        echo "Change:   +$change% ✅"
    else
        echo "Change:   $change% ⚠️"
    fi
}

handle_history() {
    if [ ! -f "$COVERAGE_HISTORY_FILE" ]; then
        echo "No coverage history found."
        exit 0
    fi
    echo "Coverage History"
    echo "━━━━━━━━━━━━━━━━"
    jq -r '.history[] | "\(.date) (\(.commit[0:7])): \(.coverage)%"' "$COVERAGE_HISTORY_FILE"
}

handle_default() {
    local current_coverage=$(get_current_coverage)
    echo "Current test coverage: $current_coverage%"
    update_history "$current_coverage"
    echo "Coverage saved to history."
}

# --- Argument Parsing ---

case "$1" in
    --diff)
        handle_diff
        ;;
    --history)
        handle_history
        ;;
    *)
        handle_default
        ;;
esac
