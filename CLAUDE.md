# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Overview

**core-agent** is a monorepo of Claude Code plugins for the Host UK federated monorepo. It contains multiple focused plugins that can be installed individually or together.

## Plugins

| Plugin | Description | Install |
|--------|-------------|---------|
| **code** | Core development - hooks, scripts, data collection | `claude plugin add host-uk/core-agent/claude/code` |
| **review** | Code review automation | `claude plugin add host-uk/core-agent/claude/review` |
| **verify** | Work verification | `claude plugin add host-uk/core-agent/claude/verify` |
| **qa** | Quality assurance loops | `claude plugin add host-uk/core-agent/claude/qa` |
| **ci** | CI/CD integration | `claude plugin add host-uk/core-agent/claude/ci` |

Or install all via marketplace:
```bash
claude plugin add host-uk/core-agent
```

## Repository Structure

```
core-agent/
├── .claude-plugin/
│   └── marketplace.json     # Plugin registry (enables auto-updates)
├── claude/
│   ├── code/                # Core development plugin
│   │   ├── .claude-plugin/
│   │   │   └── plugin.json
│   │   ├── hooks.json
│   │   ├── hooks/
│   │   ├── scripts/
│   │   ├── commands/        # /code:remember, /code:yes
│   │   ├── skills/          # Data collection skills
│   │   └── collection/      # Collection event hooks
│   ├── review/              # Code review plugin
│   │   ├── .claude-plugin/
│   │   │   └── plugin.json
│   │   └── commands/        # /review:review
│   ├── verify/              # Verification plugin
│   │   ├── .claude-plugin/
│   │   │   └── plugin.json
│   │   └── commands/        # /verify:verify
│   ├── qa/                  # QA plugin
│   │   ├── .claude-plugin/
│   │   │   └── plugin.json
│   │   ├── scripts/
│   │   └── commands/        # /qa:qa, /qa:fix
│   └── ci/                  # CI plugin
│       ├── .claude-plugin/
│       │   └── plugin.json
│       └── commands/        # /ci:ci, /ci:workflow
├── CLAUDE.md
└── .gitignore
```

## Plugin Commands

### code
- `/code:remember <fact>` - Save context that persists across compaction
- `/code:yes <task>` - Auto-approve mode with commit requirement

### review
- `/review:review [range]` - Code review on staged changes or commits

### verify
- `/verify:verify [--quick|--full]` - Verify work is complete

### qa
- `/qa:qa` - Iterative QA fix loop (runs until all checks pass)
- `/qa:fix <issue>` - Fix a specific QA issue

### ci
- `/ci:ci [status|run|logs|fix]` - CI status and management
- `/ci:workflow <type>` - Generate GitHub Actions workflows

## Core CLI Philosophy

**Always use `core` CLI instead of raw commands.** The `core` binary handles the full E2E development lifecycle for Go and PHP ecosystems.

### Command Mappings

| Instead of... | Use... |
|---------------|--------|
| `go test` | `core go test` |
| `go build` | `core build` |
| `go fmt` | `core go fmt` |
| `golangci-lint` | `core go lint` |
| `composer test` | `core php test` |
| `./vendor/bin/pint` | `core php fmt` |
| `./vendor/bin/phpstan` | `core php stan` |

### Key Commands

```bash
# Development
core dev health              # Status across repos
core dev work                # Full workflow: status → commit → push

# Go
core go test                 # Run tests
core go qa                   # Full QA pipeline

# PHP
core php test                # Run Pest tests
core php qa                  # Full QA pipeline

# Building
core build                   # Auto-detect and build

# AI
core ai task                 # Auto-select a task
core ai task:pr              # Create PR for task
```

## code Plugin Features

### Hooks

| Hook | File | Purpose |
|------|------|---------|
| PreToolUse | `prefer-core.sh` | Block dangerous commands, enforce `core` CLI |
| PostToolUse | `php-format.sh` | Auto-format PHP |
| PostToolUse | `go-format.sh` | Auto-format Go |
| PostToolUse | `check-debug.sh` | Warn about debug statements |
| PreCompact | `pre-compact.sh` | Save state before compaction |
| SessionStart | `session-start.sh` | Restore context on startup |

### Blocked Patterns

**Destructive operations:**
- `rm -rf` / `rm -r` (except node_modules, vendor, .cache)
- `mv`/`cp` with wildcards
- `xargs` with rm/mv/cp
- `find -exec` with file operations
- `sed -i` (in-place editing)

**Raw commands (use core instead):**
- `go test/build/fmt/mod` → `core go *`
- `composer test` → `core php test`

### Data Collection Skills

| Skill | Purpose |
|-------|---------|
| `ledger-papers/` | 91+ distributed ledger whitepapers |
| `project-archaeology/` | Dead project excavation |
| `bitcointalk/` | Forum thread archival |
| `coinmarketcap/` | Historical price data |
| `github-history/` | Repository history preservation |

## Development

### Adding a new plugin

1. Create `claude/<name>/.claude-plugin/plugin.json`
2. Add commands to `claude/<name>/commands/`
3. Register in `.claude-plugin/marketplace.json`

### Testing hooks locally

```bash
echo '{"tool_input": {"command": "rm -rf /"}}' | bash ./claude/code/hooks/prefer-core.sh
```

## Coding Standards

- **UK English**: colour, organisation, centre
- **Shell scripts**: Use `#!/bin/bash`, read JSON with `jq`
- **Hook output**: JSON with `decision` (approve/block) and optional `message`
- **License**: EUPL-1.2 CIC
