# core-claude

Claude Code plugin for the Host UK federated monorepo.

## Installation

```bash
/plugin marketplace add host-uk/core-claude
/plugin install core@core-claude
```

## Features

### Skills
- **core** - Core CLI command reference for multi-repo management
- **core-php** - PHP module patterns for Laravel packages
- **core-go** - Go package patterns for the CLI

### Commands
- `/core:remember <fact>` - Save context facts that persist across compaction

### Hooks

**Safety hooks:**
- Blocks destructive commands (`rm -rf`, `sed -i`, mass operations)
- Enforces `core` CLI over raw `go`/`php` commands
- Prevents random .md file creation

**Context preservation:**
- Saves state before auto-compact (prevents "amnesia")
- Restores recent session context on startup
- Extracts actionables from tool output

**Auto-formatting:**
- PHP files via Pint after edits
- Go files via gofmt after edits
- Warns about debug statements

## Dependencies

- [superpowers](https://github.com/anthropics/claude-plugins-official) from claude-plugins-official