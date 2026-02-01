# feat(ai): Add context fact management for Claude Code

## Summary

Add `core ai context` subcommands to capture and manage context facts that persist across compaction.

## Required Commands

```bash
core ai context add <fact> [--source=X]  # Save a context fact
core ai context list                      # List current facts
core ai context extract <output>          # Extract actionables from tool output
core ai context clear                     # Clear old context (>3h)
```

## Current Shell Scripts Being Replaced

- `claude/scripts/capture-context.sh` - Stores context facts
- `claude/scripts/extract-actionables.sh` - Extracts actionables from core CLI output

## Context Format

```json
[
  {"fact": "Use Action pattern not Service", "source": "user", "ts": 1234567890},
  {"fact": "FAIL: TestFoo", "source": "core go test", "ts": 1234567891}
]
```

## Extraction Patterns

The `extract` command should parse output from:
- `core go test` / `core go qa` / `core go lint` - Extract ERROR/WARN/FAIL lines
- `core php test` / `core php stan` - Extract FAIL/Error lines
- `core build` - Extract error/cannot/undefined lines

## Storage

- `~/.claude/sessions/context.json`
- Max 20 items, auto-clear after 3 hours of inactivity

## Example Usage

```bash
# Manual fact
core ai context add "User prefers UK English" --source=user

# Extract from piped output
core go test 2>&1 | core ai context extract

# List facts
core ai context list --json
```
