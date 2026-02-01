# feat(collect): Add event hook dispatch system

## Summary

Add `core collect dispatch` command to fire events during data collection, allowing modular hook handling.

## Required Commands

```bash
core collect dispatch <event> [args...]       # Fire an event
core collect dispatch on_url_found <url>      # URL discovered
core collect dispatch on_file_collected <file> <type>  # File downloaded
core collect dispatch on_collection_complete  # Batch finished
core collect hooks list                       # List registered hooks
core collect hooks register <event> <handler> # Register new hook
```

## Current Shell Script Being Replaced

- `claude/collection/dispatch.sh` - Hook dispatcher

## Events

| Event | Trigger | Args |
|-------|---------|------|
| `on_url_found` | URL discovered during collection | url |
| `on_file_collected` | File successfully downloaded | file, type |
| `on_collection_complete` | Job batch finishes | - |

## Hook Registration

Hooks defined in `hooks.json`:
```json
{
  "hooks": {
    "on_url_found": [
      {
        "name": "whitepaper-collector",
        "pattern": "\\.pdf$",
        "handler": "collect papers queue",
        "priority": 10,
        "enabled": true
      }
    ]
  }
}
```

## Pattern Matching

- Hooks can specify regex pattern for filtering
- Only matching events trigger the handler
- Multiple hooks can handle same event (priority ordering)

## Handler Types

1. **Core commands**: `collect papers queue`
2. **Shell scripts**: `./my-handler.sh`
3. **External binaries**: `/usr/local/bin/handler`

## Output Format

```json
{
  "event": "on_url_found",
  "args": ["https://example.com/paper.pdf"],
  "handlers_fired": 2,
  "results": [
    {"handler": "whitepaper-collector", "status": "ok"},
    {"handler": "archive-notifier", "status": "ok"}
  ]
}
```
