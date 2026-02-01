# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Overview

**core-agent** is the advanced in-house Claude Code plugin for the Host UK federated monorepo. The public version lives at `core-claude`.

This repository contains:
- Claude Code hooks, commands, and automation scripts
- Data collection skills for archiving OSS project data across platforms (since 2019)

## Core CLI Philosophy

**Always use `core` CLI instead of raw commands.** The `core` binary is a Go framework and CLI that handles the full E2E development lifecycle for Go and PHP ecosystems.

Why this matters:
- Every agent using `core` is testing the framework
- Missing features get raised as issues on `host-uk/core`
- Code as if functionality exists (TDD approach)
- The CLI becomes battle-tested and eventually bulletproof

### Command Mappings

| Instead of... | Use... |
|---------------|--------|
| `go test` | `core go test` |
| `go build` | `core build` |
| `go fmt` | `core go fmt` |
| `go mod tidy` | `core go mod tidy` |
| `golangci-lint` | `core go lint` |
| `composer test` | `core php test` |
| `./vendor/bin/pint` | `core php fmt` |
| `./vendor/bin/phpstan` | `core php stan` |
| `php artisan serve` | `core php dev` |
| `git status` (multi-repo) | `core dev health` |
| `git commit` (multi-repo) | `core dev commit` |
| `git push` (multi-repo) | `core dev push` |
| `gh pr create` | `core ai task:pr` |

### Key Commands

```bash
# Development workflow
core dev health              # Quick status across all repos
core dev work                # Full workflow: status → commit → push
core dev commit              # Claude-assisted commits
core dev apply --command     # Run command across repos (agent-safe)

# Go development
core go test                 # Run tests
core go test --filter=Name   # Single test
core go cov                  # Coverage report
core go fmt                  # Format code
core go lint                 # Lint with golangci-lint
core go qa                   # Full QA pipeline

# PHP development
core php test                # Run Pest tests
core php fmt                 # Format with Pint
core php stan                # PHPStan analysis
core php qa                  # Full QA pipeline
core php dev                 # Start dev server

# Building & releases
core build                   # Auto-detect and build
core build --targets=...     # Cross-compile
core ci --we-are-go-for-launch  # Publish release

# AI integration
core ai tasks                # List available tasks
core ai task                 # Auto-select a task
core ai task:commit          # Commit with task reference
core ai task:pr              # Create PR for task
core ai task:complete        # Mark task done

# Quality & security
core qa health               # Aggregate CI health
core security alerts         # All security alerts
core security deps           # Dependabot alerts
core doctor                  # Check environment
```

### Missing Features?

If `core` doesn't have what you need:

1. **Raise an issue** on `host-uk/core` describing the feature
2. **Code as if it exists** - write the call you wish existed
3. **Write a TDD test** for the expected behaviour
4. The feature will get implemented and your code will work

## Repository Structure

```
core-agent/
└── claude/
    ├── hooks/                 # Claude Code hooks
    │   ├── hooks.json         # Hook definitions
    │   └── prefer-core.sh     # PreToolUse: enforce core CLI
    ├── scripts/               # Automation scripts
    │   ├── pre-compact.sh     # Save state before compaction
    │   ├── session-start.sh   # Restore context on startup
    │   ├── php-format.sh      # Auto-format PHP after edits
    │   ├── go-format.sh       # Auto-format Go after edits
    │   └── check-debug.sh     # Warn about debug statements
    ├── commands/
    │   └── remember.md        # /core:remember command
    ├── collection/            # Data collection event hooks
    │   ├── hooks.json         # Collection hook registration
    │   ├── dispatch.sh        # Hook dispatcher
    │   └── *.sh               # Event handlers
    └── skills/                # Data collection skills
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
| PreToolUse | `prefer-core.sh` | Block dangerous commands, enforce `core` CLI |
| PostToolUse | `php-format.sh` | Auto-format PHP via `core php fmt` |
| PostToolUse | `go-format.sh` | Auto-format Go via `core go fmt` |
| PostToolUse | `check-debug.sh` | Warn about debug statements |
| PreCompact | `pre-compact.sh` | Save state before compaction |
| SessionStart | `session-start.sh` | Restore context on startup |

### Blocked Patterns

The plugin blocks dangerous patterns and enforces `core` CLI:

**Destructive operations:**
- `rm -rf` / `rm -r` (except node_modules, vendor, .cache)
- `mv`/`cp` with wildcards
- `xargs` with rm/mv/cp
- `find -exec` with file operations
- `sed -i` (in-place editing)
- `grep -l | ...` (mass file targeting)

**Raw commands (use core instead):**
- `go test/build/fmt/mod` → `core go *`
- `golangci-lint` → `core go lint`
- `composer test` → `core php test`
- `./vendor/bin/pint` → `core php fmt`
- `php artisan serve` → `core php dev`

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

Archive of 91+ distributed ledger whitepapers across 15 categories.

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

```
claude/skills/<name>/
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

This plugin is designed for use across the Host UK federated monorepo. It enforces the `core` CLI for multi-repo operations. See `/Users/snider/Code/host-uk/CLAUDE.md` for full monorepo documentation.

The `core` CLI source lives at `host-uk/core` - raise issues there for missing features.
