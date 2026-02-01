# feat(hooks): Auto-format code silently after edits

## Summary

Add PostToolUse hooks that silently auto-format code after Write/Edit, without polluting context with formatter output.

## Current State

Current hooks call formatters but output is visible:
- `php-format.sh` - runs Pint
- `go-format.sh` - runs gofmt

This creates noise in the conversation.

## Solution

Use `suppressOutput: true` to hide formatter output entirely.

## Hook Configuration

```json
{
  "hooks": {
    "PostToolUse": [
      {
        "matcher": "Write|Edit",
        "hooks": [
          {
            "type": "command",
            "command": "core ai format-file",
            "statusMessage": "Formatting..."
          }
        ]
      }
    ]
  }
}
```

## Implementation

`core ai format-file`:

```bash
#!/bin/bash
INPUT=$(cat)
FILE=$(echo "$INPUT" | jq -r '.tool_input.file_path // empty')

[ -z "$FILE" ] || [ ! -f "$FILE" ] && exit 0

case "$FILE" in
  *.go)
    core go fmt "$FILE" 2>/dev/null
    ;;
  *.php)
    core php fmt "$FILE" 2>/dev/null
    ;;
  *.ts|*.tsx|*.js|*.jsx)
    npx prettier --write "$FILE" 2>/dev/null
    ;;
  *.json)
    jq '.' "$FILE" > "$FILE.tmp" && mv "$FILE.tmp" "$FILE" 2>/dev/null
    ;;
esac

# Suppress all output - formatting is silent
echo '{"suppressOutput": true}'
```

## Benefits

1. **No context pollution** - formatter output hidden
2. **Consistent formatting** - every file formatted on save
3. **No manual step** - Claude doesn't need to remember to format
4. **Fast** - formatters are quick, doesn't block significantly

## Error Handling

Only report if formatting fails badly:

```bash
if ! core go fmt "$FILE" 2>&1; then
  # Only surface actual errors, not warnings
  echo '{"systemMessage": "Format failed for '$FILE' - syntax error?"}'
  exit 0
fi

echo '{"suppressOutput": true}'
```
