<!-- SPDX-Licence-Identifier: EUPL-1.2 -->
---
module: dappco.re/go/agent
surface: Go binary (core-agent) + Claude Code plugin + opencode plugin + PHP platform
role: AUI — agent-facing dispatch/orchestration/fleet (lthn/desktop is the HUI twin)
---

# core/agent — RFC

> The matter-of-fact contract for the **core-agent** Go binary: what every subsystem does,
> in present tense. The code conforms to this document; `GOAL.md` gates the two into parity
> in both directions. To advance the repo, drive an implementation pass against this file.
>
> Go is the local runtime (dispatch, workspace, brain, opencode, MCP/hub). PHP is the fleet
> platform (REST API, admin UI, persistent storage, content). The contract is shared; this
> document describes the **Go** surface, and points to `php/` for the PHP body (§17).

---

## 1. Purpose

core-agent dispatches AI coding agents (Claude, Codex, Gemini, opencode) into sandboxed
containers, runs an opencode-backed agent fleet, serves an MCP + hub control plane, and
carries shared semantic memory (OpenBrain). It is the **AUI** — the agent-facing surface,
where an agent *wields* the system headlessly. `lthn/desktop` is its **HUI** twin, where a
human *drives* the same machinery interactively. Both own a full `pkg/opencode`, tailored to
their driver; the copies diverge by design and are deliberately not shared.

Every capability is a named Core action; the MCP server and the plugins expose subsets of
those actions to their hosts.

---

## 2. Binary & Modes

A single binary, `core-agent` (`dappco.re/go/agent`, built from `go/cmd/core-agent/`):

| Mode | What it does |
|------|--------------|
| `mcp` | stdio MCP server for a coding-agent host (registered by `dappco.re/go/mcp`). Default Claude Code integration. |
| `serve` | HTTP MCP daemon for cross-agent / CI / remote use. |
| `hub` | Loopback control plane: a strict-bound `coreapi.Engine` on `--http 127.0.0.1:9201` (bearer-auth) serving the opencode control + proxy groups and brain, plus a fail-closed core/mcp HTTP+SSE plane on `--mcp-http 127.0.0.1:9202`. A non-optional `pkg/audit` edge records every request. This is the surface the desktop crew and the plugins drive. |
| `chat --user=<id>` | REPL against the local LEM engine (lthn-mlx / lthn-ai driver), auto-captured to the user's portable DuckDB archive. |
| `serve-status` / `serve-reload` / `serve-profiles` | Inspect / hot-swap / list the local model engine's profiles. |
| `models-download` / `models-job` | Queue and poll Hugging Face model downloads. |
| `run flow <path>` | Execute a YAML workflow (§14). |

---

## 3. Domain Model

| Model | Purpose |
|-------|---------|
| `AgentPlan` | Structured work plan with phases. Soft-deleted, activity-logged. Status: `draft`, `active`, `in_progress`, `needs_verification`, `verified`, `completed`, `archived`. |
| `AgentPhase` | A phase within a plan — tasks, dependencies, status. |
| `AgentSession` | An agent work session — context, work_log, artefacts, handoff. |
| `AgentMessage` | Direct agent-to-agent message (chronological, not semantic). |
| `AgentApiKey` | External agent access key — hashed, scoped, rate-limited. |
| `BrainMemory` | Semantic knowledge entry — tags, type, confidence, vector-indexed, supersession chain. |
| `Issue` / `IssueComment` | Bug/feature/task tracking and comments — labels, priority, sprint. |
| `Sprint` | Time-boxed iteration grouping issues. |
| `Task` | Simple task — title, status, file/line reference. |
| `Prompt` / `PromptVersion` | Reusable AI prompt template (system + user) and its immutable snapshots. |
| `PlanTemplateVersion` | Immutable YAML plan-template snapshot. |
| `WorkspaceState` | Typed key-value state per plan, shared across sessions. |
| `Sandbox` | A running opencode container — `id`, `image`, host port, status (`running`/`stopped`), created_at. Persisted via the ORM so reconcile survives restart. |

**Relationships.** A Plan has many Phases; each Phase has tasks, dependencies, status. A
Session belongs to a Plan and an Agent and produces artefacts. BrainMemory is scoped by
workspace and agent, with supersession chains linking new knowledge to what it replaces.
Issues belong to Sprints. Each Prompt mutation creates an immutable PromptVersion.

---

## 4. Dispatch and Workspace — the doing path

```
Task → queue → concurrency + rate gate → workspace prep → container spawn → agent runs
     → completion pipeline (§5)
```

### 4.1 Workspace prep (`agentic.prep` / `agentic_prep_workspace`)

`PrepInput{Repo, Org, Task, Agent, Issue, PR, Branch, Tag, Template, PlanTemplate,
Variables, Persona, DryRun}` resolves a workspace directory under `WorkspaceRoot()`
(`~/Lethean/workspace/{org}/{repo}/{task-N | pr-N | branch | tag}`). Prep:

1. Clones the repo into `repo/` inside the workspace. The clone source is the **local
   mirror** `~/Code/{org}/{repo}` (fast; kept fresh by the post-completion sync, §11), not
   Forge directly. A re-prep of an existing workspace pulls `--ff-only` instead of cloning.
2. Creates the working branch `agent/{task-slug}`.
3. Clones workspace dependencies and copies the repo's spec tree (`plans/.../RFC*.md`) into
   `specs/`, and the org `docs` repo into `.core/reference/docs/`.
4. Builds the agent prompt (§4.2) and writes a prompt snapshot.

`PrepOutput{Success, WorkspaceDir, RepoDir, Branch, Prompt, PromptVersion, Memories,
Consumers, Resumed}`.

### 4.2 Prompt building

`buildPrompt` assembles, in order: `TASK`, `REPO/branch`, detected `LANGUAGE` / `BUILD` /
`TEST` commands, optional `PERSONA` (from `pkg/lib/persona/`), language `WORKFLOW`, the
`ISSUE` body, `CONTEXT` recalled from OpenBrain, `CONSUMERS` (modules importing this repo),
recent git log, an optional `PLAN`, and `CONSTRAINTS` (read CODEX.md/CLAUDE.md, conventional
commits with the Virgil trailer, build + test before commit).

### 4.3 Agent commands

`agentCommandResult(agent, prompt)` builds the command line per agent type (`agent` is
`base[:model]`):

| Agent | Command shape |
|-------|---------------|
| `claude` | `claude -p <prompt> --output-format text --dangerously-skip-permissions --no-session-persistence --append-system-prompt "SANDBOX: …"` `[--model]` |
| `codex` | `codex exec --dangerously-bypass-approvals-and-sandbox -o ../.meta/agent-codex.log` `[--profile <lem> | --model <model>]` `<prompt>`. `codex:review` runs a fixed review prompt. |
| `gemini` | `gemini -p <prompt> --yolo --sandbox` `[-m gemini-2.5-<model>]` |
| `coderabbit` | `coderabbit review --plain --base HEAD~1` `[--type] [--config CLAUDE.md]` |
| `opencode` | `sh -c 'OPENCODE_CONFIG_CONTENT=… opencode run --dangerously-skip-permissions --model <provider/model> [--agent] <prompt>'` (profile from §6) |
| `local` | `sh -c 'socat … host.docker.internal:11434 & codex exec … --oss --local-provider ollama -m <model> …'` (ollama bridged from host) |

The approval-bypass flags are intentional: the **container is the isolation boundary** (§6
permission boundary, §4.4), not per-tool prompts.

### 4.4 Container execution

`containerCommandFor(runtime, image, gpu, command, args, workspaceDir, metaDir)` builds the
run line. Docker, Podman and Apple Container share the flag shape (`run --rm -v … -w …`);
only the binary differs. The container:

- bind-mounts the workspace: `-v {workspaceDir}:/workspace -v {metaDir}:/workspace/.meta`,
  working directory `-w /workspace/repo`;
- mounts agent credentials read-only as needed (`~/.codex`, and `~/.claude`/`~/.gemini` for
  those agents);
- passes provider keys + git identity (`GIT_USER_NAME=Virgil`, `GIT_USER_EMAIL`) and Go
  resolution env (`GONOSUMCHECK`, `GOFLAGS`) by environment;
- on Docker/Podman adds `--add-host=host.docker.internal:host-gateway`; with GPU,
  `--gpus=all` (NVIDIA) or `--gpu=metal` (Apple, roadmap);
- runs `sh -c` with a guard (`/workspace/repo` must exist) then the agent command, then
  `chmod -R a+w` so the host can read results back.

Runtime is auto-detected in preference order **Apple Container → Docker → Podman** (Apple
Containers give hardware-VM isolation with sub-second start on macOS 26+; the default image
is `core-dev`). The choice is overridable in `agents.yaml` or per dispatch.

### 4.5 Queue, concurrency, rate

A persistent queue drains when a slot frees: concurrency limits (per pool + per model) and
rate limits (daily, min/sustained delay, burst window) gate each spawn (§15). Dispatch emits
`AgentStarted` → runs → `AgentCompleted`.

### 4.6 Outcome and the bail

`detectFinalStatus` reads the workspace after the agent exits: a non-empty `BLOCKED.md` →
status `blocked` (the agent's **free ticket out** — it stops and surfaces a question rather
than thrashing); a non-zero exit / killed process → `failed`; otherwise `completed`.
Repeated failures back a pool off (3 failures < 60s → 30-minute backoff).

---

## 5. Completion Pipeline

On `AgentCompleted`, a handler chain fires, composed as the `agent.completion` Task:

| Step | Action | Description |
|------|--------|-------------|
| 1 | `agentic.qa` | Run core/lint + build + test; capture **every** finding to the workspace DuckDB (no filtering). |
| 2 | `agentic.auto-pr` | Open a pull request from passing output. |
| 3 | `agentic.verify` | Check CI + review criteria → `PRMerged` or `PRNeedsReview`. |
| 4 (async) | `agentic.ingest` | Extract findings → Forge issues. |
| 5 (async) | `agentic.poke` | Drain the queue — dispatch the next waiting task. |
| 6 (async) | `agentic.commit` | Workspace DuckDB → go-store journal. |

QA captures raw findings; intelligence comes from analysis *after*, not filtering during.
Before commit, Poindexter clusters the findings in N-dimensional space (tool, severity,
file, category, frequency) and diffs against prior cycles to surface new / resolved /
persistent findings into `.meta/report.json`. The aggregated summary is journalled; the raw
DuckDB is then purged.

---

## 6. opencode — the AUI surface

core-agent **owns** opencode. `pkg/opencode` is tailored for agent-driven use; the desktop
copy is tailored for human-driven use (same machinery, divergent surface, not shared).

### 6.1 Two roles

- **Generate** — drive a model through a sandboxed opencode session as an inference proxy:
  `GenerateInput{Prompt, Profile, Model, Agent, SandboxID}` → ensure a running sandbox →
  `POST /session` → `POST /session/:id/message` → read the assistant text. The
  `ProviderManager` (`agentic/opencode.go`) registers this as the real backend behind every
  provider name, so generation is in-process — no HTTP hop inside core-agent.
- **Doing-slice** — mount a prepped workspace (§4) into the opencode container so opencode
  codes against a ready-to-go project. The HUI attaches a human (web / TUI); the AUI drives
  headless via the session API.

### 6.2 Service lifecycle

`Service.Start(profile)` spawns `<runtime> run -d -p 127.0.0.1:{hostPort}:4096 -e
OPENCODE_CONFIG_CONTENT=… -e OPENCODE_SERVER_PASSWORD=… --label {installID} {image} opencode
web --hostname 0.0.0.0 --port 4096`, allocates a host port from the ephemeral range with a
bounded retry, persists a `Sandbox` record, registers the reverse-proxy target, waits for
`/global/health`, then applies the profile via `PATCH /global/config`. `Stop` cancels the
SSE subscription, removes the container, marks the record `Stopped`, drops the proxy target.
`Reconcile` adopts only containers carrying this install's label.

### 6.3 Profiles

A profile names the upstream provider + model + base URL for a sandbox.
`opencodeProfileConfig` maps profile names to local / free-compute endpoints — e.g.
`gemma4-agentic` → `core-local` `google/gemma-4-26B-A4B-it` @ `:8001`; `lemma` → `:8006`;
`qwen36` → `:8003`; `core-mlx` / `core-vllm` variants across `:8001-:8011`; small-model
companions per profile. Every field is overridable by `CORE_OPENCODE_{PROFILE}_{KEY}` env.
`opencodeConfigContent` renders the opencode wire config (provider block, model, tool
allow-list, permission map).

### 6.4 Permission boundary follows the driver

opencode permissions are `allow | ask | deny`, granular (`"bash": {"git *": "allow", "rm *":
"deny"}`), per-agent-overridable. **AUI runs all-allow** — the container is the isolation
boundary, which is why dispatch passes approval-bypass flags. **HUI runs `ask`**, human in
the loop. A headless run that must answer an "ask" responds via `POST
/session/:id/permissions/:permissionID` against a policy (the SSE stream carries the prompt);
nothing blocks.

### 6.5 Session API (the control surface)

opencode-serve exposes the full surface the hub fronts and proxies: `POST /session`,
`GET|DELETE|PATCH /session/:id`, `/children`, `/abort`, `/fork`; `POST /session/:id/message`
(sync, single-shot) and **`POST /session/:id/prompt_async`** (no-wait — the fleet primitive);
`POST /session/:id/permissions/:id`; SSE **`/global/event`** (progress feed); `GET|PATCH
/config`, `GET /config/providers`; **`POST /mcp`** (attach an MCP server at runtime); `/agent`,
`/command`, `/global/health`. Auth is HTTP Basic (`OPENCODE_SERVER_PASSWORD`); the hub adds
bearer at its edge. `prompt_async` + the SSE stream is how many sessions run concurrently —
the fleet engine.

### 6.6 Hub edge

The `hub` mode (§2) is the SASE access edge for opencode: a strict-bound loopback engine
with bearer auth and a non-optional audit sink wraps the opencode control + proxy groups, so
opencode itself (which runs in a sandbox and does not audit itself) is audited at the edge.
See `docs/RFC.serve.md`.

---

## 7. Plugin Providers — Claude Code + opencode

core-agent ships plugins that expose its capabilities to a coding-agent host. Two providers,
one capability set, **shared assets from one source**:

- **`provider/claude/`** — Claude Code plugin: MCP server (`mcp.json`), hooks (`hooks.json` —
  inbox notifications, auto-format), agents, commands, skills.
- **`provider/opencode/`** — opencode plugin (`@opencode-ai/plugin`): capabilities as custom
  `tool()` exports (`dispatch`, `status`, `scan`, `brain_recall`, …); event hooks
  (`session.idle` → done, `session.error` → BLOCKED, `tool.execute.after` → progress) feed
  §12's report-home loop; the ctx `client` SDK interacts with the running session.

**Personas ≡ opencode agent definitions.** Personas map onto opencode agent files (markdown
frontmatter: `description`, `mode: primary|subagent`, `model`, `prompt`, per-tool
`permission`). Cerberus = a permission-tuned `subagent`. **Skills ≡ opencode skills**
(`SKILL.md` + the `skill` tool). **Dispatch is two-layer:** opencode-native (the `Task` tool
spawns subagents as child sessions, in-session) **+** core-agent's cross-host fleet (the
`dispatch` custom tool spawns containers across free compute). A session can also be handed
core-agent's tools by attaching the hub MCP plane via `POST /mcp` — a route alternative to
the custom-tool exports.

Every opencode instance on the free-compute fleet loads this plugin → is fleet-capable
(dispatch + recall + report) → the orchestrator starts/steers the fleet and watches progress
via §12.

---

## 8. Brain — OpenBrain

Shared semantic knowledge. Capabilities: `brain.remember`, `brain.recall`, `brain.forget`,
`brain.list`, plus agent-to-agent messaging (§12). Go is the local bridge (`pkg/brain`,
`agentic/brain_client.go`); PHP holds the persistent store — MariaDB `brain_memories`
(source of truth: workspace_id, agent_id, type, content, tags, confidence, supersedes_id,
expires_at), Qdrant vectors (768d, nomic-embed-text via Ollama, cosine), filtered semantic
search. `brain_remember` stores → embeds → upserts; `brain_recall` embeds the query →
searches Qdrant → hydrates from MariaDB. Memories are never hard-deleted (soft-delete +
supersession + TTL + confidence ranking).

---

## 9. Forge

Forge (Gitea/Forgejo) integration via `forge_client.go` / `transport.go`:
`issue.{get,list,create,update,comment,archive}`, `pr.{get,list,merge,close}`,
`branch.delete`, `scan` (repos for actionable-label issues: agentic, help-wanted, bug),
`mirror` (Forge → GitHub). Agent branches (`agent/*`) are ephemeral and deleted after merge
or close to keep workspace prep clean.

---

## 10. Session and Plan Lifecycle

`session.start(plan, agent)` → the agent appends to `work_log` → `session.continue(id, work)`
→ `session.end(id, summary, handoff)`; `session.handoff` and `session.replay` recover context
for the next agent. Plans (`plan.{create,read,update,list,delete}`) have Phases
(`phase.{get,update_status,add_checkpoint}`) which have Tasks
(`task.{create,update,toggle}`). `WorkspaceState` (`state.{set,get,list,delete}`) is a typed
key-value store shared between sessions within a plan — Agent A writes, Agent B reads later.
Plans and templates are versioned; YAML plan templates render via `template.*`.

---

## 11. Fleet and Remote Sync — lthn.ai

**Fleet mode** connects to `api.lthn.ai` with an `AgentApiKey` (bootstrapped by
`agent.auth.login` exchanging a 6-digit pairing code). It registers capabilities, receives
jobs over SSE (polling fallback `GET /v1/fleet/task/next` for NAT'd nodes), heartbeats, and
reports results. Anyone running core-agent contributes compute.

**Remote sync** pushes the local `.core/db.duckdb` dispatch history + findings to PHP
(`agent.sync.push` → `POST /v1/agent/sync` → BrainMemory embeddings + WorkspaceState) and
pulls fleet-wide context (`agent.sync.pull` ← `GET /v1/agent/context`). Unreachable API →
results queue in `db.duckdb` with backoff (1s → 5min) and flush on reconnect. No API key =
fully offline; sync is additive, never required.

---

## 12. Channels and Notifications — the report-home loop

`message.send` / `message.inbox` / `message.conversation` carry direct agent-to-agent
messages (`commands_message.go`, `message.go`). A push listener surfaces new messages
(`InboxMessage` IPC) and dispatched-agent progress back to the orchestrator through the
Claude / opencode plugins — the loop that lets the fleet report to Cladius from inside
Claude Code.

> NB: this loop is currently out of action and needs restoring. GOAL.md tracks it as a known
> gap until the notification path (push listener → plugin surface) is live again.

---

## 13. Content Generation

PHP-driven; the Go surface is `content.generate` / `content.batch`. Product briefs (per
service) → versioned, categorised prompt templates (content / development / visual / system)
→ AI generation → drafts → quality refinement → publication. Natural-Progression SEO
schedules content revisions 8–62 minutes after a Googlebot visit so updates read as organic.
SEO schema (`content.schema.generate`) emits Article / FAQ / HowTo JSON-LD.

---

## 14. Flows

Declarative YAML workflows under `pkg/lib/flow/`, path-addressed (path = semantics) and
composable (a flow calls flows via `flow:`). Sequential pipelines, parallel fan-out,
conditional steps (`when:`), agent-dispatch steps, manual approval gates. Run with
`core-agent run flow <path.yaml> [--dry-run] [--var k=v]`. See `docs/flow/RFC.md`.

---

## 15. Configuration

`agents.yaml`:

- **dispatch**: `default_agent`, `default_template`, `workspace_root`, `runtime`
  (`auto|apple|docker|podman`), `image`, `gpu`.
- **concurrency**: per pool, with per-model sub-limits (e.g. `claude.{total,opus,sonnet,
  haiku}`).
- **rates**: per pool — `daily_limit`, `min_delay`, `sustained_delay`, `burst_window`,
  `burst_delay`.
- **agents**: named identities — `host`, `runner`, `roles`.

Named identities: `cladius` (local, claude, dispatch/review/plan), `charon` (remote, claude,
worker/review), `codex` (cloud, openai, worker), `clotho` (local, claude, review/qa). Codex
model variants are selected with `agent: codex:{model}` (`gpt-5.4` frontier … `gpt-5.4-mini`,
`gpt-5.3-codex`, `gpt-5.3-codex-spark`, etc.).

---

## 16. State Persistence — go-store

`.core/db.duckdb` holds top-level state in three groups: `queue` (`{repo}/{branch}` → task,
agent, status, priority — survives restart), `concurrency` (`{agent-type}` → running count —
no over-dispatch after restart), `registry` (`{org}/{repo}/{workspace}` → status, PID, agent,
branch — no ghost agents). On startup the registry is restored and any `running` entry whose
PID is dead is reaped to `failed`. Each workspace gets its own DuckDB for the dispatch cycle
(events, findings); on cleanup, stats are written to the parent `.core/workspace/db.duckdb`
**before** the workspace dir is deleted, so "what happened in the last 50 dispatches?" is a
query, not a directory scan. If go-store is not loaded, all state falls back to in-memory
maps — no crashes, no hard dependency.

---

## 17. Polyglot Mapping

Go is the local MCP server (dispatch, workspace, brain, opencode); PHP is the web platform
(REST API, admin UI, persistent storage, content generation). Capabilities map 1:1 —
`pkg/brain/*` ↔ `Actions/Brain/*`, `pkg/agentic/dispatch.go` ↔
`Console/Commands/DispatchCommand`, `pkg/agentic/actions.go` ↔ `Mcp/Tools/*`, SQLite/file ↔
MariaDB. The PHP body lives in `php/` and `docs/php-agent/RFC.md`; this document does not
duplicate it.

---

## 18. Reference

| Resource | Location |
|----------|----------|
| AX principles | `docs/RFC-CORE-008-AGENT-EXPERIENCE.md` |
| Hub / serve edge | `docs/RFC.serve.md` |
| Autonomous pipeline | `docs/RFC-AGENT-PIPELINE.md` |
| Fleet topology | `docs/RFC-AGENT-TOPOLOGY.md` |
| Flows | `docs/flow/RFC.md` |
| Plugins | `docs/plugins/RFC.md`, `docs/RFC-AGENT-PLUGIN-RESTRUCTURE.md` |
| PHP implementation | `php/`, `docs/php-agent/RFC.md` |
| Implementation goal / gate | `GOAL.md` |

> The repo `docs/` tree holds the detailed sub-specs this document consolidates. Where a
> sub-spec and this RFC disagree, the code is the tie-breaker (GOAL.md reconciles both
> directions); fold genuine detail up into the relevant section here rather than leaving
> drifting duplicates.
