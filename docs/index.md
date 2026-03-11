---
title: Core Agent
description: AI agent orchestration, Claude Code plugins, and lifecycle management for the Host UK platform — a polyglot Go + PHP repository.
---

# Core Agent

Core Agent (`forge.lthn.ai/core/agent`) is a polyglot repository containing **Go libraries**, **CLI commands**, **MCP servers**, and a **Laravel PHP package** that together provide AI agent orchestration for the Host UK platform.

It answers three questions:

1. **How do agents get work?** -- The lifecycle package manages tasks, dispatching, and quota enforcement. The PHP side exposes a REST API for plans, sessions, and phases.
2. **How do agents run?** -- The dispatch and jobrunner packages poll for work, clone repositories, invoke Claude/Codex/Gemini, and report results back to Forgejo.
3. **How do agents collaborate?** -- Sessions, plans, and the OpenBrain vector store enable multi-agent handoff, replay, and persistent memory.


## Quick Start

### Go (library / CLI commands)

The Go module is `forge.lthn.ai/core/agent`. It requires Go 1.26+.

```bash
# Run tests
core go test

# Full QA pipeline
core go qa
```

Key CLI commands (registered into the `core` binary via `cli.RegisterCommands`):

| Command | Description |
|---------|-------------|
| `core ai tasks` | List available tasks from the agentic API |
| `core ai task [id]` | View or claim a specific task |
| `core ai task --auto` | Auto-select the highest-priority pending task |
| `core ai agent list` | List configured AgentCI dispatch targets |
| `core ai agent add <name> <host>` | Register a new agent machine |
| `core ai agent fleet` | Show fleet status from the agent registry |
| `core ai dispatch watch` | Poll the PHP API for work and execute phases |
| `core ai dispatch run` | Process a single ticket from the local queue |

### PHP (Laravel package)

The PHP package is `lthn/agent` (Composer name). It depends on `lthn/php` (the foundation framework).

```bash
# Run tests
composer test

# Fix code style
composer lint
```

The package auto-registers via Laravel's service provider discovery (`Core\Mod\Agentic\Boot`).


## Package Layout

### Go Packages

| Package | Path | Purpose |
|---------|------|---------|
| `lifecycle` | `pkg/lifecycle/` | Core domain: tasks, agents, dispatcher, allowance quotas, events, API client, brain (OpenBrain), embedded prompts |
| `loop` | `pkg/loop/` | Autonomous agent loop: prompt-parse-execute cycle with tool calling against any `inference.TextModel` |
| `orchestrator` | `pkg/orchestrator/` | Clotho protocol: dual-run verification, agent configuration, security helpers |
| `jobrunner` | `pkg/jobrunner/` | Poll-dispatch engine: `Poller`, `Journal`, Forgejo source, pipeline handlers |
| `plugin` | `pkg/plugin/` | Plugin contract tests |
| `workspace` | `pkg/workspace/` | Workspace contract tests |

### Go Commands

| Directory | Registered As | Purpose |
|-----------|---------------|---------|
| `cmd/tasks/` | `core ai tasks`, `core ai task` | Task listing, viewing, claiming, updating |
| `cmd/agent/` | `core ai agent` | AgentCI machine management (add, list, status, setup, fleet) |
| `cmd/dispatch/` | `core ai dispatch` | Work queue processor (runs on agent machines) |
| `cmd/workspace/` | `core workspace task`, `core workspace agent` | Isolated git-worktree workspaces for task execution |
| `cmd/taskgit/` | *(internal)* | Git operations for task branches |
| `cmd/mcp/` | Standalone binary | MCP server (stdio) with marketplace, ethics, and core CLI tools |

### MCP Servers

| Directory | Transport | Tools |
|-----------|-----------|-------|
| `cmd/mcp/` | stdio (mcp-go) | `marketplace_list`, `marketplace_plugin_info`, `core_cli`, `ethics_check` |
| `google/mcp/` | HTTP (:8080) | `core_go_test`, `core_dev_health`, `core_dev_commit` |

### Claude Code Plugins

| Plugin | Path | Commands |
|--------|------|----------|
| **code** | `claude/code/` | `/code:remember`, `/code:yes`, `/code:qa` |
| **review** | `claude/review/` | `/review:review`, `/review:security`, `/review:pr` |
| **verify** | `claude/verify/` | `/verify:verify`, `/verify:ready`, `/verify:tests` |
| **qa** | `claude/qa/` | `/qa:qa`, `/qa:fix` |
| **ci** | `claude/ci/` | `/ci:ci`, `/ci:workflow`, `/ci:fix`, `/ci:run`, `/ci:status` |

Install all plugins: `claude plugin add host-uk/core-agent`

### Codex Plugins

The `codex/` directory mirrors the Claude plugin structure for OpenAI Codex, plus additional plugins for ethics, guardrails, performance, and issue management.

### PHP Package

| Directory | Namespace | Purpose |
|-----------|-----------|---------|
| `src/php/` | `Core\Mod\Agentic\` | Laravel service provider, models, controllers, services |
| `src/php/Actions/` | `...\Actions\` | Single-purpose business logic (Brain, Forge, Phase, Plan, Session, Task) |
| `src/php/Controllers/` | `...\Controllers\` | REST API controllers for go-agentic client consumption |
| `src/php/Models/` | `...\Models\` | Eloquent models: AgentPlan, AgentPhase, AgentSession, AgentApiKey, BrainMemory, Task, Prompt, WorkspaceState |
| `src/php/Services/` | `...\Services\` | AgenticManager (multi-provider), BrainService (Ollama+Qdrant), ForgejoService, Claude/Gemini/OpenAI services |
| `src/php/Mcp/` | `...\Mcp\` | MCP tool implementations: Brain, Content, Phase, Plan, Session, State, Task, Template |
| `src/php/View/` | `...\View\` | Livewire admin components (Dashboard, Plans, Sessions, ApiKeys, Templates, ToolAnalytics) |
| `src/php/Migrations/` | | 10 database migrations |
| `src/php/tests/` | | Pest test suite |


## Dependencies

### Go

| Dependency | Purpose |
|------------|---------|
| `forge.lthn.ai/core/go` | DI container and service lifecycle |
| `forge.lthn.ai/core/cli` | CLI framework (cobra + bubbletea TUI) |
| `forge.lthn.ai/core/go-ai` | AI meta-hub (MCP facade) |
| `forge.lthn.ai/core/go-config` | Configuration management (viper) |
| `forge.lthn.ai/core/go-inference` | TextModel/Backend interfaces |
| `forge.lthn.ai/core/go-io` | Filesystem abstraction |
| `forge.lthn.ai/core/go-log` | Structured logging |
| `forge.lthn.ai/core/go-ratelimit` | Rate limiting primitives |
| `forge.lthn.ai/core/go-scm` | Source control (Forgejo client, repo registry) |
| `forge.lthn.ai/core/go-store` | Key-value store abstraction |
| `forge.lthn.ai/core/go-i18n` | Internationalisation |
| `github.com/mark3labs/mcp-go` | Model Context Protocol SDK |
| `github.com/redis/go-redis/v9` | Redis client (registry + allowance backends) |
| `modernc.org/sqlite` | Pure-Go SQLite (registry + allowance backends) |
| `codeberg.org/mvdkleijn/forgejo-sdk` | Forgejo API SDK |

### PHP

| Dependency | Purpose |
|------------|---------|
| `lthn/php` | Foundation framework (events, modules, lifecycle) |
| `livewire/livewire` | Admin panel reactive components |
| `pestphp/pest` | Testing framework |
| `orchestra/testbench` | Laravel package testing |


## Configuration

### Go Client (`~/.core/agentic.yaml`)

```yaml
base_url: https://api.lthn.sh
token: your-api-token
default_project: my-project
agent_id: cladius
```

Environment variables override the YAML file:

| Variable | Purpose |
|----------|---------|
| `AGENTIC_BASE_URL` | API base URL |
| `AGENTIC_TOKEN` | Authentication token |
| `AGENTIC_PROJECT` | Default project |
| `AGENTIC_AGENT_ID` | Agent identifier |

### PHP (`.env`)

```env
ANTHROPIC_API_KEY=sk-ant-...
GOOGLE_AI_API_KEY=...
OPENAI_API_KEY=sk-...
```

The agentic module also reads `BRAIN_DB_*` for the dedicated brain database connection and Ollama/Qdrant URLs from `mcp.brain.*` config keys.


## Licence

EUPL-1.2
