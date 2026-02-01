# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Overview

**core-agent** contains Claude Code plugins and data collection skills for the Host UK federated monorepo. It has two main components:

1. **claude/** - Claude Code plugin with hooks, commands, and automation scripts
2. **claude-cowork/** - Data collection skills for archiving blockchain/cryptocurrency research

## Repository Structure

```
core-agent/
├── claude/                    # Claude Code plugin
│   ├── hooks/hooks.json       # Hook definitions
│   ├── hooks/prefer-core.sh   # PreToolUse: block dangerous commands
│   ├── scripts/               # Automation scripts
│   │   ├── pre-compact.sh     # Save state before compaction
│   │   ├── session-start.sh   # Restore context on startup
│   │   ├── php-format.sh      # Auto-format PHP after edits
│   │   ├── go-format.sh       # Auto-format Go after edits
│   │   └── check-debug.sh     # Warn about debug statements
│   └── commands/
│       └── remember.md        # /core:remember command
│
└── claude-cowork/             # Data collection skills
    ├── hooks/                 # Collection event hooks
    │   ├── hooks.json         # Hook registration
    │   └── dispatch.sh        # Hook dispatcher
    └── skills/                # Collection skills
        ├── ledger-papers/     # Whitepaper archive (91+ papers)
        ├── project-archaeology/ # Dead project excavation
        ├── bitcointalk/       # BitcoinTalk thread collection
        ├── coinmarketcap/     # Market data collection
        ├── github-history/    # Git history preservation
        └── ...                # Other collectors
```

## Claude Plugin Features

### Hooks

| Hook | File | Purpose |
|------|------|---------|
| PreToolUse | `prefer-core.sh` | Block destructive commands, enforce `core` CLI |
| PostToolUse | `php-format.sh` | Auto-format PHP files after edits |
| PostToolUse | `go-format.sh` | Auto-format Go files after edits |
| PostToolUse | `check-debug.sh` | Warn about debug statements |
| PreCompact | `pre-compact.sh` | Save state before compaction |
| SessionStart | `session-start.sh` | Restore context on startup |

### Blocked Commands

The plugin blocks these patterns to prevent accidental damage:

- `rm -rf` / `rm -r` (except node_modules, vendor, .cache)
- `mv`/`cp` with wildcards
- `xargs` with rm/mv/cp
- `find -exec` with file operations
- `sed -i` (in-place editing)
- `grep -l | ...` (mass file targeting)
- Raw `go` commands (use `core go *`)
- Raw `php artisan` / `composer test` (use `core php *`)

### Commands

- `/core:remember <fact>` - Save context that persists across compaction

### Context Preservation

State is saved to `~/.claude/sessions/` before compaction:
- Working directory and branch
- Git status (modified files)
- In-progress todos
- User-saved context facts

## Data Collection Skills

### ledger-papers

Archive of 91+ distributed ledger whitepapers across 15 categories (genesis, cryptonote, MRL, privacy, smart-contracts, etc).

```bash
./discover.sh --all              # List all papers
./discover.sh --category=privacy # Filter by category
```

### project-archaeology

Excavates abandoned CryptoNote projects before data is lost.

```bash
./excavate.sh masari             # Full dig
./excavate.sh masari --scan-only # Check what's accessible
```

### Other collectors

- `bitcointalk/` - BitcoinTalk thread archival
- `coinmarketcap/` - Historical price data
- `github-history/` - Repository history preservation
- `wallet-releases/` - Binary release archival
- `block-explorer/` - Blockchain data indexing

## Development

### Testing hooks locally

```bash
# Simulate PreToolUse hook input
echo '{"tool_input": {"command": "rm -rf /"}}' | bash ./claude/hooks/prefer-core.sh
```

### Adding new hooks

1. Add script to `claude/scripts/`
2. Register in `claude/hooks/hooks.json`
3. Test with simulated input

### Collection skill structure

Each skill follows this pattern:

```
skills/<name>/
├── SKILL.md           # Documentation
├── discover.sh        # Job generator (outputs URL|FILENAME|TYPE|METADATA)
├── process.sh         # Job processor (optional)
└── registry.json      # Data registry (optional)
```

## Coding Standards

- **UK English**: colour, organisation, centre
- **Shell scripts**: Use `#!/bin/bash`, read JSON with `jq`
- **Hook output**: JSON with `decision` (approve/block) and optional `message`
- **License**: EUPL-1.2 CIC

## Integration with Host UK

This plugin is designed for use across the Host UK federated monorepo. It enforces the `core` CLI for multi-repo operations instead of raw git/go/php commands. See the parent `/Users/snider/Code/host-uk/CLAUDE.md` for full monorepo documentation.
