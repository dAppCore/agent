#!/bin/bash
# Enforce strict type hints in PHP files.

read -r input
FILE_PATH=$(echo "$input" | jq -r '.tool_input.file_path // empty')

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

if [[ -n "$FILE_PATH" && -f "$FILE_PATH" ]]; then
    php "${SCRIPT_DIR}/check-types.php" "$FILE_PATH"
fi

# Pass through the input
echo "$input"
