# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Overview

**core-agent** is a polyglot monorepo (Go + PHP) for AI agent orchestration. The Go side handles agent-side execution, CLI commands, and autonomous agent loops. The PHP side (Laravel package `lthn/agent`) provides the backend API, persistent storage, multi-provider AI services, and admin panel. They communicate via REST API.

The repo also contains Claude Code plugins (5), Codex plugins (13), a Gemini CLI extension, and two MCP servers.

## Core CLI — Always Use It

**Never use raw `go`, `php`, or `composer` commands.** The `core` CLI wraps both toolchains and is enforced by PreToolUse hooks that will block violations.

| Instead of... | Use... |
|---------------|--------|
| `go test` | `core go test` |
| `go build` | `core build` |
| `go fmt` | `core go fmt` |
| `go vet` | `core go vet` |
| `golangci-lint` | `core go lint` |
| `composer test` / `./vendor/bin/pest` | `core php test` |
| `./vendor/bin/pint` / `composer lint` | `core php fmt` |
| `./vendor/bin/phpstan` | `core php stan` |
| `php artisan serve` | `core php dev` |

## Build & Test Commands

```bash
# Go
core go test                                        # Run all Go tests
core go test --run TestMemoryRegistry_Register_Good # Run single test
core go qa                                          # Full QA: fmt + vet + lint + test
core go qa full                                     # QA + race detector + vuln scan
core go cov                                         # Test coverage
core build                                          # Verify Go packages compile

# PHP
core php test                                       # Run Pest suite
core php test --filter=AgenticManagerTest            # Run specific test file
core php fmt                                        # Format (Laravel Pint)
core php stan                                       # Static analysis (PHPStan)
core php qa                                         # Full PHP QA pipeline

# MCP servers (standalone builds)
cd cmd/mcp && go build -o agent-mcp .               # Stdio MCP server
cd google/mcp && go build -o google-mcp .           # HTTP MCP server (port 8080)

# Workspace
make setup                                          # Full bootstrap (deps + core + clone repos)
core dev health                                     # Status across repos
```

## Architecture

```
                Forgejo
                  |
         [ForgejoSource polls]
                  |
                  v
+-- Go: jobrunner Poller --+      +-- PHP: Laravel Backend --+
| ForgejoSource            |      | AgentApiController       |
| DispatchHandler ---------|----->| /v1/plans                |
| CompletionHandler        |      | /v1/sessions             |
| ResolveThreadsHandler    |      | /v1/plans/*/phases       |
+--------------------------+      +-------------+------------+
                                                |
                                        [Eloquent models]
                                        AgentPlan, AgentPhase,
                                        AgentSession, BrainMemory
```

### Go Packages (`pkg/`)

- **`lifecycle/`** — Core domain layer. Task, AgentInfo, Plan, Phase, Session types. Agent registry (Memory/SQLite/Redis backends), task router (capability matching + load scoring), allowance system (quota enforcement), dispatcher (orchestrates dispatch with exponential backoff), event system, brain (vector store), context (git integration).
- **`loop/`** — Autonomous agent reasoning engine. Prompt-parse-execute cycle against any `inference.TextModel` with tool calling and streaming.
- **`orchestrator/`** — Clotho protocol for dual-run verification and agent orchestration.
- **`jobrunner/`** — Poll-dispatch engine for agent-side work execution. Polls Forgejo for work items, executes phases, reports results.

### Go Commands (`cmd/`)

- **`tasks/`** — `core ai tasks`, `core ai task [id]` — task management
- **`agent/`** — `core ai agent` — agent machine management (add, list, status, fleet)
- **`dispatch/`** — `core ai dispatch` — work queue processor (watch, run)
- **`workspace/`** — `core workspace task`, `core workspace agent` — git worktree isolation
- **`mcp/`** — Standalone stdio MCP server exposing `marketplace_list`, `marketplace_plugin_info`, `core_cli`, `ethics_check`

### PHP (`src/php/`)

- **Namespace**: `Core\Mod\Agentic\` (service provider: `Boot`)
- **Models/** — 19 Eloquent models (AgentPlan, AgentPhase, AgentSession, BrainMemory, Task, Prompt, etc.)
- **Services/** — AgenticManager (multi-provider: Claude/Gemini/OpenAI), BrainService (Ollama+Qdrant), ForgejoService, AI services with stream parsing and retry traits
- **Controllers/** — AgentApiController (REST endpoints)
- **Actions/** — Single-purpose action classes (Brain, Forge, Phase, Plan, Session, Task)
- **View/** — Livewire admin panel components (Dashboard, Plans, Sessions, ApiKeys, Templates, Playground, etc.)
- **Mcp/** — MCP tool implementations (Brain, Content, Phase, Plan, Session, State, Task, Template)
- **Migrations/** — 10 migrations (run automatically on boot)

## Claude Code Plugins (`claude/`)

Five plugins installable individually or via marketplace:

| Plugin | Commands |
|--------|----------|
| **code** | `/code:remember`, `/code:yes`, `/code:qa` |
| **review** | `/review:review`, `/review:security`, `/review:pr`, `/review:pipeline` |
| **verify** | `/verify:verify`, `/verify:ready`, `/verify:tests` |
| **qa** | `/qa:qa`, `/qa:fix`, `/qa:check`, `/qa:lint` |
| **ci** | `/ci:ci`, `/ci:workflow`, `/ci:fix`, `/ci:run`, `/ci:status` |

### Hooks (code plugin)

**PreToolUse**: `prefer-core.sh` blocks destructive operations (`rm -rf`, `sed -i`, `xargs rm`, `find -exec rm`, `grep -l | ...`, `mv/cp *`) and raw go/php commands. `block-docs.sh` prevents random `.md` file creation.

**PostToolUse**: Auto-formats Go (`gofmt`) and PHP (`pint`) after edits. Warns about debug statements (`dd()`, `dump()`, `fmt.Println()`).

**PreCompact**: Saves session state. **SessionStart**: Restores session context.

## Other Directories

- **`codex/`** — 13 Codex plugins mirroring Claude structure plus ethics, guardrails, perf, issue, coolify, awareness
- **`agents/`** — 13 specialist agent categories (design, engineering, marketing, product, testing, etc.) with example configs and system prompts
- **`google/gemini-cli/`** — Gemini CLI extension (TypeScript, `npm run build`)
- **`google/mcp/`** — HTTP MCP server exposing `core_go_test`, `core_dev_health`, `core_dev_commit`
- **`docs/`** — `architecture.md` (deep dive), `development.md` (comprehensive dev guide), `docs/plans/` (design documents)
- **`scripts/`** — Environment setup scripts (`install-core.sh`, `install-deps.sh`, `agent-runner.sh`, etc.)

## Testing Conventions

### Go

Uses `testify/assert` and `testify/require`. Name tests with suffixes:
- `_Good` — happy path
- `_Bad` — expected error conditions
- `_Ugly` — panics and edge cases

Use `require` for preconditions (stops on failure), `assert` for verifications (reports all failures).

### PHP

Pest with Orchestra Testbench. Feature tests use `RefreshDatabase`. Helpers: `createWorkspace()`, `createApiKey($workspace, ...)`.

## Coding Standards

- **UK English**: colour, organisation, centre, licence, behaviour
- **Go**: standard `gofmt`, errors via `core.E("scope.Method", "what failed", err)`
- **PHP**: `declare(strict_types=1)`, full type hints, PSR-12 via Pint, Pest syntax for tests
- **Shell**: `#!/bin/bash`, JSON input via `jq`, output `{"decision": "approve"|"block", "message": "..."}`
- **Commits**: conventional — `type(scope): description` (e.g. `feat(lifecycle): add exponential backoff`)
- **Licence**: EUPL-1.2 CIC

## Prerequisites

| Tool | Version | Purpose |
|------|---------|---------|
| Go | 1.26+ | Go packages, CLI, MCP servers |
| PHP | 8.2+ | Laravel package, Pest tests |
| Composer | 2.x | PHP dependencies |
| `core` CLI | latest | Wraps Go/PHP toolchains (enforced by hooks) |
| `jq` | any | JSON parsing in shell hooks |

Go module is `forge.lthn.ai/core/agent`, participates in a Go workspace (`go.work`) resolving all `forge.lthn.ai/core/*` dependencies locally.
