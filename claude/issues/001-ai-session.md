# feat(ai): Add session state management for Claude Code hooks

## Summary

Add `core ai session` subcommands to manage Claude Code session state, replacing shell scripts that currently handle this.

## Required Commands

```bash
core ai session save              # Save current session state (pre-compact)
core ai session restore           # Restore session state (session-start)
core ai session stats             # Track tool calls, suggest compaction
core ai session clear             # Clear stale session data
```

## Current Shell Scripts Being Replaced

- `claude/scripts/pre-compact.sh` - Saves state before auto-compact
- `claude/scripts/session-start.sh` - Restores context on startup
- `claude/scripts/suggest-compact.sh` - Suggests compaction at intervals

## State to Manage

- Working directory and git branch
- Git status (modified files)
- In-progress todos
- Context facts (decisions, actionables)
- Tool call counter per session

## Storage

State stored in `~/.claude/sessions/`:
- `scratchpad.md` - Human-readable resume state
- `context.json` - Structured context facts
- `stats.json` - Session statistics

## Output Format

```json
{
  "saved": true,
  "path": "~/.claude/sessions/scratchpad.md",
  "facts": 5,
  "tool_calls": 47
}
```
