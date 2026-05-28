---
title: Core Agent
description: AI agent orchestration for the Core ecosystem — a single Go binary that runs as an MCP server (stdio + HTTP) and a CLI for dispatch, fleet sync, OpenBrain memory, and local-model chat.
---

# Core Agent

Core Agent (`dappco.re/go/agent`) is a single Go binary that orchestrates AI agents across the Core ecosystem. It runs as an **MCP server** — stdio for IDE integration, HTTP for cross-agent communication — and ships a **CLI** for everything from dispatching a ticket to a sandboxed worker through to chatting with a local model.

The binary ships under two names: `core-agent` (legacy) and `lthn-agent` (the `lthn-{mlx,cuda,amd,agent}` family naming). It detects its invocation name from `argv[0]` and identifies accordingly in version output, banners, and admin-token prefixes. Either build name produces the same behaviour.

It answers three questions:

1. **How do agents get work?** -- the `agentic` package exposes MCP dispatch tools (`agentic_dispatch`, `agentic_scan`, `agentic_create_epic`, the plan/phase/session surface) and CLI verbs that fan a tracked issue out to a sandboxed runner.
2. **How do agents run?** -- dispatch preps an isolated workspace, spawns the chosen runner (Claude / Codex / Gemini / OpenCode against a local model), watches it to completion, and drives the closeout pipeline (QA → auto-PR → verify → merge).
3. **How do agents collaborate?** -- OpenBrain (`brain` package) gives durable memory + cross-agent messaging; sessions, plans, and handoff notes let one agent pick up where another stopped.

## Quick Start

The Go module is `dappco.re/go/agent`. It requires Go 1.26+ and lives in the `go/` subdirectory of the repository.

```bash
cd go
go build ./cmd/core-agent/        # build the binary
go install ./cmd/core-agent/      # install to $GOPATH/bin
go test ./... -count=1            # run the test suite
```

Cross-compile for Charon (the homelab Linux box):

```bash
cd go
GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -o core-agent-linux ./cmd/core-agent/
```

## Binary Modes

| Invocation | What it does |
|------------|--------------|
| `core-agent mcp` | MCP server over stdio — the transport an IDE (Claude Code etc.) connects to. |
| `core-agent serve` | HTTP MCP daemon — cross-agent communication, CI, the homelab fleet. |
| `core-agent chat --user=<id>` | Interactive REPL against a local `lthn-mlx` serve, auto-captured to the user's portable chat archive. |
| `core-agent serve-status` / `serve-reload` / `serve-profiles` | Inspect and hot-swap the local `lthn-mlx` model engine via its `/v1/admin/*` API. |
| `core-agent models-download` / `models-job` | Queue and poll Hugging Face model downloads on the local engine. |
| `core-agent version` / `check` / `env` | Version + build info, workspace/dependency health check, resolved environment keys. |

The `mcp` and `serve` commands are provided by the shared `dappco.re/go/mcp` service the binary registers; the rest are registered directly by `cmd/core-agent`.

## Go Packages

| Package | Path | Purpose |
|---------|------|---------|
| `agentic` | `pkg/agentic/` | The orchestration core: MCP dispatch tools, prep/verify/scan, fleet + platform sync, the plan/phase/session command surface, mirror to GitHub. |
| `brain` | `pkg/brain/` | OpenBrain client — remember / recall / forget / list and cross-agent messaging, both in-process and over `/v1/brain/*`. |
| `lemma` | `pkg/lemma/` | Client for the local `lthn-mlx` model engine: chat sessions, the `/v1/admin/*` control surface, model downloads. |
| `chathistory` | `pkg/chathistory/` | Per-user portable DuckDB chat archive (continuity rights — the file is the user's property). |
| `monitor` | `pkg/monitor/` | Background agent monitoring, completion tracking, repo sync. |
| `runner` | `pkg/runner/` | Local + container runners that execute a dispatched agent. |
| `setup` | `pkg/setup/` | Project-type detection and `.core/` workspace scaffolding. |
| `lib` | `pkg/lib/` | Embedded personas, prompt + flow templates, and workspace scaffolds (`flow`, `persona`, `prompt`, `task`, `workspace`). |
| `messages` | `pkg/messages/` | Typed IPC message definitions for the dispatch pipeline. |
| `agentcompat` | `pkg/agentcompat/` | Compatibility shims for agent-tooling interop. |

## MCP Tool Surface

The `agentic` and `brain` subsystems register the bulk of the tool surface. Highlights:

| Category | Tools |
|----------|-------|
| Dispatch | `agentic_dispatch`, `agentic_dispatch_remote`, `agentic_dispatch_start`, `agentic_dispatch_shutdown`, `agentic_status_remote` |
| Workspace | `agentic_prep_workspace`, `agentic_resume`, `agentic_watch` |
| PR / review | `agentic_create_pr`, `agentic_list_prs`, `agentic_create_epic`, `agentic_review_queue` |
| Mirror / scan | `agentic_mirror` (Forge → GitHub), `agentic_scan` (Forge issues) |
| Plans / phases / sessions | `agentic_plan_*`, `agentic_phase_*`, `agentic_session_*` |
| Brain | `brain_remember`, `brain_recall`, `brain_forget`, `brain_list` |
| Messaging | `agent_send`, `agent_inbox`, `agent_conversation` |
| Local model | `lemma_send` (chat with the local model, auto-captured to the caller's archive) |

## Repository Layout

```
agent/
├── go/                  Go module — module path: dappco.re/go/agent
│   ├── cmd/core-agent/  Binary entry point — builds core-agent or lthn-agent
│   └── pkg/             agentic, brain, lemma, chathistory, monitor, runner, setup, lib, messages, agentcompat
├── php/                 Laravel package (Core\Mod\Agentic\*) for the hosted lthn.ai service
├── provider/            Per-provider integrations: claude/ (Claude Code plugins), codex/, google/, hermes/
├── scripts/            Install + local-inference launch helpers (gemma4/qwen36 stacks, local-agent.sh)
├── docs/               This documentation tree
├── external/            Dev-workspace submodules for dappco.re/go/* dependencies
└── vm/                  Containerised dev stack
```

## Dependencies

| Dependency | Purpose |
|------------|---------|
| `dappco.re/go` | DI container, service lifecycle, core primitives (`core.E`, `core.Result`, `c.Process()`, `c.Fs()`). |
| `dappco.re/go/mcp` | MCP service — registers the `mcp` (stdio) and `serve` (HTTP) commands and the tool-recording harness. |
| `github.com/modelcontextprotocol/go-sdk` | Model Context Protocol SDK. |

The authoritative `dappco.re/go/*` dependency snapshot is `module-graph.json` at the repository root.

## Licence

EUPL-1.2
