#!/bin/bash
# Auto-format Go files after edits using core go fmt
# Policy: HIDE success (formatting is silent background operation)

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/output-policy.sh"

read -r input
FILE_PATH=$(echo "$input" | jq -r '.tool_input.file_path // empty')

if [[ -n "$FILE_PATH" && -f "$FILE_PATH" ]]; then
    # Run gofmt/goimports on the file silently
    if command -v core &> /dev/null; then
        core go fmt --fix "$FILE_PATH" 2>/dev/null || true
    elif command -v goimports &> /dev/null; then
        goimports -w "$FILE_PATH" 2>/dev/null || true
    elif command -v gofmt &> /dev/null; then
        gofmt -w "$FILE_PATH" 2>/dev/null || true
    fi
fi

# Silent success - no output needed
hide_success
