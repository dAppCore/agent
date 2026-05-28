---
title: Architecture
description: Internal architecture of core/agent — the Go binary's dispatch pipeline, runner, monitor, OpenBrain, local-model lanes, and the PHP backend that backs the hosted service.
---

# Architecture

Core Agent is a single Go binary (`dappco.re/go/agent`, built from `go/cmd/core-agent`) that runs as an MCP server and CLI. A separate PHP/Laravel package (`Core\Mod\Agentic\*`) provides the hosted-service backend at `lthn.ai` — REST API, persistent storage, multi-provider AI services, and the admin panel. The two collaborate through `/v1/*` HTTP endpoints.

The binary is built on the `dappco.re/go` DI container. `main.go` constructs a `core.New(...)` with a set of services and lets the CLI framework dispatch commands:

```go
core.New(
    core.WithOption("name", "core-agent"),
    core.WithService(agentic.ProcessRegister),
    core.WithService(agentic.Register),     // dispatch tools + IPC pipeline
    core.WithService(runner.Register),       // agent execution
    core.WithService(monitor.Register),      // monitoring + repo sync
    core.WithService(brain.Register),        // OpenBrain memory + messaging
    core.WithService(setup.Register),        // workspace scaffolding
    core.WithService(registerLemmaSubsystem),// local-model MCP tool
    core.WithService(coremcp.Register),      // mcp + serve commands, tool harness
)
```

`coremcp.Register` (from `dappco.re/go/mcp`) is what supplies the `mcp` (stdio) and `serve` (HTTP) commands; the agentic, brain, and lemma subsystems register their MCP tools into that service.

## Go: Orchestration (`pkg/agentic/`)

`agentic` is the orchestration core. It registers the dispatch MCP tools and, via `RegisterHandlers`, wires the closeout IPC pipeline. On registration it loads `agents.yaml` and enables the pipeline stages by default:

```go
c.Config().Enable("auto-qa")     // run QA after the agent completes
c.Config().Enable("auto-pr")     // open a PR when QA passes
c.Config().Enable("auto-merge")  // verify + merge the PR
c.Config().Enable("auto-ingest") // file issues from findings
```

### Dispatch

`agentic_dispatch` takes a `DispatchInput` (repo, task, agent, template, persona, issue/PR, branch/tag, dry-run) and:

1. Preps a sandboxed workspace for the task.
2. Resolves the runner command from the agent string (`agentCommand`). Native agents (`claude`, `coderabbit`, `opencode`) run on the host; others (`codex`, `gemini`) run inside Docker.
3. Spawns the agent process and returns a `DispatchOutput` (workspace dir, PID, output file).

Agent strings carry an optional model after a colon — `codex:gpt-5.4-mini`, `claude:opus`, `opencode:gemma4-mlx-agentic`. For the local OpenCode lanes see [`local-inference.md`](local-inference.md) and [`local-inference-typologies.md`](local-inference-typologies.md).

### Closeout pipeline

Once the agent finishes, completion is detected and the typed IPC pipeline (`pkg/messages/`) runs the stages:

```
AgentCompleted → QA → AutoPR → Verify → Merge
```

Each stage is gated by its `auto-*` config flag, so an operator can disable any stage. Findings can be ingested back into the tracker as issues.

### Remote dispatch

`agentic_dispatch_remote` and `agentic_status_remote` proxy a dispatch to another `core-agent` instance over its HTTP MCP endpoint (the homelab fleet path). `agentic_dispatch_start` / `agentic_dispatch_shutdown` control the dispatch queue lifecycle — run `dispatch_start` after a restart to unfreeze the queue.

### Plans, phases, sessions

The package also exposes the structured-work surface as both MCP tools and CLI commands (with `agentic:` aliases): `plan/*`, `phase/*`, and `session/*`. Plans hold ordered phases; sessions track an agent's work with a log, artefacts, and handoff notes for the next agent. These are persisted via the PHP `/v1/plans`, `/v1/plans/{slug}/phases`, and `/v1/sessions` endpoints.

### Fleet + platform sync

`agentic` registers fleet machines and syncs repos against `agents.yaml`. Fleet registration posts to `/v1/fleet/register` through a TLS-validating shared HTTP client (`transport.go`'s `defaultClient`).

## Go: Runner (`pkg/runner/`)

`runner` executes dispatched agents and tracks their workspaces. It holds a `core.Registry[*WorkspaceStatus]`, a dispatch lock, a drain lock, and per-agent backoff/fail counters. It uses `c.Lock(name)` for named mutexes when the Core container is present, falling back to channel locks for standalone use. The queue (`queue.go`) drains pending work; `paths.go` centralises workspace path resolution.

## Go: Monitor (`pkg/monitor/`)

`monitor` runs background monitoring: it harvests completion signals (`harvest.go`), exposes a monitor API (`monitor.go`), and keeps ecosystem repos in sync (`sync.go`).

## Go: OpenBrain (`pkg/brain/`)

`brain` is the OpenBrain client — durable memory plus cross-agent messaging. It exposes MCP tools (`brain_remember`, `brain_recall`, `brain_forget`, `brain_list`) and the messaging tools (`agent_send`, `agent_inbox`, `agent_conversation`). Two transport modes exist:

- **Direct** (`direct.go`) — calls `/v1/brain/*` on the API through the shared `dappco.re/go/mcp/.../brain/client`, with Bearer auth, default-org injection, `~/.claude/brain.key` (`0600`) handling, absolute-URL rejection, retry with jitter, and a circuit breaker.
- **Bridge** (`provider.go`) — forwards to the IDE bridge over WebSocket; recall/list return empty synchronously and deliver results async (by design for the bridge path).

The canonical map of every Brain call site, its protections, and its request/response shapes lives in [`BRAIN-CALLERS.md`](BRAIN-CALLERS.md).

## Go: Local model (`pkg/lemma/` + `pkg/chathistory/`)

`lemma` is the client for the local `lthn-mlx` model engine. It provides chat sessions, the `/v1/admin/*` control surface (`admin.go` — status, reload, profiles, model downloads), and is exposed two ways:

- The `chat` CLI command opens a REPL against the engine.
- The `lemma_send` MCP tool lets a calling agent send a message and get a reply.

Both auto-capture every turn into the caller's portable archive via `chathistory`, a per-user DuckDB file at `~/Lethean/data/users/<id>/chats.duckdb`. The file is the user's property (continuity rights): a model or provider change can never take the chat history away. `export.go` handles export; `migrations/` carries the schema.

## Go: Setup (`pkg/setup/`)

`setup` detects a project's type (Go, Wails, PHP, Node, …) and scaffolds a `.core/` directory with `build.yaml` + `test.yaml`, optionally extracting a workspace template from `pkg/lib`.

## Go: Library (`pkg/lib/`)

`lib` holds embedded assets and the helpers that extract them: `persona/` (domain personas), `prompt/` (prompt templates), `task/` (task templates including code review + simplifier), `flow/` (per-language flow definitions plus the `upgrade/` YAML flows), and `workspace/` (workspace scaffolds — `default`, `review`, `security`). `ExtractWorkspace` and `ListWorkspaces` are the entry points used by `setup`.

## PHP: Backend (`php/`)

The PHP package backs the hosted service. It registers via Laravel's event-driven module lifecycle (`Boot`) and is organised into:

- `Actions/` — single-purpose business logic, grouped by domain (Auth, Brain, Credits, Fleet, Forge, Issue, Phase, Plan, Session, Sprint, Subscription, Sync, Task).
- `Controllers/Api/` — REST controllers behind `AgentApiAuth` (Bearer tokens, scope-based permissions, workspace binding).
- `Models/` — Eloquent models (AgentPlan, AgentPhase, AgentSession, BrainMemory, …), multi-tenant via `BelongsToWorkspace`.
- `Services/` — provider services (Claude, Gemini, OpenAI) behind a manager, plus `BrainService`.
- `Mcp/` — server-side MCP tool implementations.
- `View/` — Livewire admin components.
- `Migrations/` — schema.

### BrainService (OpenBrain)

`BrainService` is the canonical PHP write/read path behind the controller, MCP tools, console commands, and the Livewire explorer. It writes to MariaDB first and queues async indexing (`EmbedMemory`) into Qdrant + Elasticsearch; recall embeds the query, searches Qdrant, then hydrates rows from MariaDB. Memories are workspace-scoped, with `org` and `project` filters. Qdrant access is authenticated via an `api-key` header.

## Data Flow: End-to-End Dispatch

1. A tracked issue is scanned (`agentic_scan`) or a dispatch is requested directly.
2. `agentic_dispatch` preps an isolated workspace and resolves the runner.
3. The runner (Claude / Codex / Gemini / OpenCode) makes changes, commits, and pushes.
4. Completion is detected; the IPC pipeline runs QA → auto-PR → verify → merge, each gated by its `auto-*` flag.
5. Findings can be ingested back into the tracker as issues.
6. For cross-machine work, the dispatch is proxied to a remote `core-agent` over HTTP MCP, and status is polled with `agentic_status_remote`.
