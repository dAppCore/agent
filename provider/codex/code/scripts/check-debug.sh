#!/bin/bash
# Warn about debug statements left in code after edits
# Policy: EXPOSE warning when found, HIDE when clean

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/output-policy.sh"

read -r input
FILE_PATH=$(echo "$input" | jq -r '.tool_input.file_path // empty')

FOUND=""

if [[ -n "$FILE_PATH" && -f "$FILE_PATH" ]]; then
    case "$FILE_PATH" in
        *.go)
            FOUND=$(grep -n "fmt\.Println\|log\.Println" "$FILE_PATH" 2>/dev/null | head -3)
            ;;
        *.php)
            FOUND=$(grep -n "dd(\|dump(\|var_dump(\|print_r(" "$FILE_PATH" 2>/dev/null | head -3)
            ;;
    esac
fi

if [[ -n "$FOUND" ]]; then
    expose_warning "Debug statements in \`$FILE_PATH\`" "\`\`\`\n$FOUND\n\`\`\`"
else
    pass_through "$input"
fi
