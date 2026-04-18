#!/bin/bash
# Auto-format PHP files after edits using core php fmt
# Policy: HIDE success (formatting is silent background operation)

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/output-policy.sh"

read -r input
FILE_PATH=$(echo "$input" | jq -r '.tool_input.file_path // empty')

if [[ -n "$FILE_PATH" && -f "$FILE_PATH" ]]; then
    # Run Pint on the file silently
    if command -v core &> /dev/null; then
        core php fmt --fix "$FILE_PATH" 2>/dev/null || true
    elif [[ -f "./vendor/bin/pint" ]]; then
        ./vendor/bin/pint "$FILE_PATH" 2>/dev/null || true
    fi
fi

# Silent success - no output needed
hide_success
