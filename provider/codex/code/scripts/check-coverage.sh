#!/bin/bash
# Check for a drop in test coverage.
# Policy: EXPOSE warning when coverage drops, HIDE when stable/improved

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/output-policy.sh"

# Source the main coverage script to use its functions
source claude/code/commands/coverage.sh 2>/dev/null || true

read -r input

# Get current and previous coverage (with fallbacks)
CURRENT_COVERAGE=$(get_current_coverage 2>/dev/null || echo "0")
PREVIOUS_COVERAGE=$(get_previous_coverage 2>/dev/null || echo "0")

# Compare coverage
if awk -v current="$CURRENT_COVERAGE" -v previous="$PREVIOUS_COVERAGE" 'BEGIN {exit !(current < previous)}'; then
    DROP=$(awk -v c="$CURRENT_COVERAGE" -v p="$PREVIOUS_COVERAGE" 'BEGIN {printf "%.1f", p - c}')
    expose_warning "Test coverage dropped by ${DROP}%" "Previous: ${PREVIOUS_COVERAGE}% → Current: ${CURRENT_COVERAGE}%"
else
    pass_through "$input"
fi
