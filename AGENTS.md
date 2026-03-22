# AGENTS.md — Core Agent

This file provides guidance to Codex when working with code in this repository.

## Project Overview

Core Agent (`dappco.re/go/agent`) is the agent orchestration platform for the Core ecosystem. It provides an MCP server binary (`core-agent`) with tools for dispatching subagents, workspace management, cross-agent messaging, OpenBrain integration, and monitoring.

## Architecture

```
cmd/main.go        — Binary entry point, Core CLI (no cobra)
pkg/agentic/       — Dispatch, workspace prep, status, queue, plans, PRs, epics
pkg/brain/         — OpenBrain knowledge store (direct HTTP + IDE bridge)
pkg/monitor/       — Background monitoring, harvest, sync
pkg/lib/           — Embedded prompts, tasks, flows, personas, workspace templates
pkg/setup/         — Project detection, config generation, scaffolding
```

## Conventions

This project follows the **AX (Agent Experience)** design principles from RFC-025.

### Code Style
- **UK English**: colour, organisation, initialise (never American spellings)
- **Errors**: `core.E("operation", "message", err)` — never `fmt.Errorf`
- **Logging**: `core.Error/Info/Warn/Debug` — never `log.*` or `fmt.Print*`
- **Filesystem**: `core.Fs{}` with `Result` returns — never `os.ReadFile/WriteFile`
- **Strings**: `core.Contains/Split/Trim/HasPrefix/Sprintf` — never `strings.*` or `fmt.Sprintf`
- **Returns**: `core.Result{Value, OK}` — never `(value, error)` pairs
- **Comments**: Usage examples showing HOW with real values, not descriptions
- **Names**: Predictable, unabbreviated (Config not Cfg, Service not Srv)
- **Imports**: stdlib `io` aliased as `goio`
- **Interface checks**: `var _ Interface = (*Impl)(nil)` compile-time assertions

### Build & Test
```bash
go build ./...
go test ./...
go vet ./...
```

### Branch Strategy
- Work on `dev` branch, never push to `main` directly
- PRs required for `main` — Codex review gate
- Commit format: `type(scope): description`
- Co-author: `Co-Authored-By: Virgil <virgil@lethean.io>`

### Dependencies
- Only `dappco.re/go/core` for primitives (fs, errors, logging, strings)
- Domain packages: `process`, `ws`, `mcp` for actual services
- No `go-io`, `go-log`, `cli` — Core provides these natively
- Use `go get -u ./...` for dependency updates, never manual go.mod edits

## MCP Tools

The binary exposes these MCP tools when run as `core-agent mcp`:

| Tool | Purpose |
|------|---------|
| `agentic_dispatch` | Dispatch subagent to sandboxed workspace |
| `agentic_status` | List workspace statuses |
| `agentic_resume` | Resume blocked/failed workspace |
| `agentic_prep_workspace` | Prepare workspace without dispatching |
| `agentic_create_pr` | Create PR from workspace |
| `agentic_list_prs` | List PRs across repos |
| `agentic_create_epic` | Create epic with child issues |
| `agentic_scan` | Scan Forge for actionable issues |
| `agentic_plan_*` | Plan CRUD (create, read, update, delete, list) |
| `brain_recall` | Semantic search OpenBrain |
| `brain_remember` | Store to OpenBrain |
| `brain_forget` | Remove from OpenBrain |
| `agent_send` | Send message to another agent |
| `agent_inbox` | Read inbox messages |
| `metrics_record` | Record metrics event |
| `metrics_query` | Query metrics |
