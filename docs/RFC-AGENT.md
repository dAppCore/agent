---
module: core/agent
repo: core/agent
lang: multi
tier: consumer
depends:
  - code/core/go/process
  - code/core/go/store
  - code/core/mcp
  - code/snider/poindexter
tags:
  - dispatch
  - orchestration
  - pipeline
  - agents
  - memory
---

# core/agent RFC — Agentic Dispatch, Orchestration, and Pipeline Management

> The cross-cutting contract for the agent system.
> An agent should be able to understand the full agent architecture from this document alone.
> Both Go and PHP implementations conform to this contract.

**Sub-specs:** [Pipeline](RFC.pipeline.md) | [Topology](RFC.topology.md) | [Plugin Restructure](RFC.plugin-restructure.md)

---

## 1. Purpose

core/agent dispatches AI agents (Claude, Codex, Gemini) to work on tasks in sandboxed git worktrees, monitors their progress, verifies output, and manages the merge pipeline. It provides a shared semantic memory (OpenBrain), inter-agent messaging, Forge integration, and fleet-scale orchestration.

The contract is language-agnostic. Go implements the local MCP server and dispatch binary. PHP implements the web platform, admin UI, and persistent storage. Both expose the same capabilities through their native surfaces (MCP tools in Go, REST API + MCP tools in PHP).

---

## 2. Domain Model

| Model | Purpose |
|-------|---------|
| `AgentPlan` | Structured work plan with phases, soft-deleted, activity-logged. Status enum: `draft`, `active`, `in_progress`, `needs_verification`, `verified`, `completed`, `archived`. Both Go and PHP must accept all values. |
| `AgentPhase` | Individual phase within a plan (tasks, dependencies, status) |
| `AgentSession` | Agent work session (context, work_log, artefacts, handoff) |
| `AgentMessage` | Direct agent-to-agent messaging (chronological, not semantic) |
| `AgentApiKey` | External agent access key (hashed, scoped, rate-limited) |
| `BrainMemory` | Semantic knowledge entry (tags, confidence, vector-indexed) |
| `Issue` | Bug/feature/task tracking (labels, priority, sprint) |
| `IssueComment` | Comment on an issue |
| `Sprint` | Time-boxed iteration grouping issues |
| `Task` | Simple task (title, status, file/line reference) |
| `Prompt` | Reusable AI prompt template (system + user template) |
| `PromptVersion` | Immutable prompt snapshot |
| `PlanTemplateVersion` | Immutable YAML template snapshot |
| `WorkspaceState` | Key-value state per plan (typed, shared across sessions) |

### Relationships

- A **Plan** has many **Phases**. Each Phase has tasks, dependencies, and a status.
- A **Session** belongs to a Plan and an Agent. Sessions track work_log and produce artefacts.
- **BrainMemory** entries are scoped by workspace and agent. Supersession chains link newer knowledge to what it replaces.
- **Issues** belong to Sprints. Agents scan Issues for actionable work.
- **Prompts** are versioned — each mutation creates an immutable **PromptVersion**.

---

## 3. Capabilities

Both implementations provide these capabilities, registered as named actions:

### Dispatch and Workspace

| Capability | Description |
|------------|-------------|
| `dispatch` | Dispatch an agent to a sandboxed workspace |
| `prep` | Prepare a workspace (clone, branch, install deps) |
| `status` | Query workspace status across all active agents |
| `resume` | Resume a paused or failed agent session |
| `scan` | Scan Forge repos for actionable issues |
| `watch` | Watch workspace for agent output changes |
| `complete` | Run the full completion pipeline (QA → PR → Verify → Ingest → Poke) |

### Pipeline

| Capability | Description |
|------------|-------------|
| `qa` | Run quality checks on agent output |
| `auto-pr` | Create a pull request from agent output |
| `verify` | Verify PR passes CI and review criteria |
| `ingest` | Extract findings from agent output and create issues |
| `poke` | Drain the dispatch queue (trigger next queued task) |
| `mirror` | Mirror changes to secondary remotes |

### Forge

| Capability | Description |
|------------|-------------|
| `issue.get` | Get a single Forge issue |
| `issue.list` | List Forge issues with filtering |
| `issue.create` | Create a Forge issue |
| `pr.get` | Get a single pull request |
| `pr.list` | List pull requests |
| `pr.merge` | Merge a pull request |
| `pr.close` | Close a pull request without merging |
| `branch.delete` | Delete a feature branch after merge or close |

### Brain

| Capability | Description |
|------------|-------------|
| `brain.remember` | Store knowledge with tags and embedding |
| `brain.recall` | Semantic search across stored knowledge |
| `brain.forget` | Remove a memory entry |
| `brain.list` | List memories with filtering |

### Session and Messaging

| Capability | Description |
|------------|-------------|
| `session.start` | Start an agent session within a plan |
| `session.continue` | Resume a session with new work |
| `session.end` | End a session with summary and handoff |
| `message.send` | Send a message to another agent |
| `message.inbox` | Read incoming messages |
| `message.conversation` | Get conversation thread with a specific agent |

### Plans

| Capability | Description |
|------------|-------------|
| `plan.create` | Create a structured work plan |
| `plan.read` | Read a plan by ID or slug |
| `plan.update` | Update plan status |
| `plan.list` | List plans with filtering |
| `plan.delete` | Archive (soft-delete) a plan |

### Review and Epic

| Capability | Description |
|------------|-------------|
| `review-queue` | List items awaiting human review |
| `epic` | Create an epic spanning multiple repos/plans |

---

## 4. OpenBrain Architecture

Shared semantic knowledge store. All agents read and write via `brain_*` tools.

### Storage Layers

| Layer | Technology | Purpose |
|-------|-----------|---------|
| Relational | MariaDB `brain_memories` | Source of truth — workspace_id, agent_id, type, tags, content, confidence |
| Vector | Qdrant `openbrain` collection | 768d vectors (nomic-embed-text via Ollama), cosine distance, filtered search |
| Embedding | Ollama (nomic-embed-text) | Generates vectors from memory content |

### brain_memories Schema

| Column | Type | Purpose |
|--------|------|---------|
| `id` | UUID | Primary key and Qdrant point ID |
| `workspace_id` | FK | Multi-tenant isolation |
| `agent_id` | string | Who wrote it (cladius, charon, codex, lem) |
| `type` | enum | decision, observation, convention, research, plan, bug, architecture |
| `content` | text | The knowledge (markdown) |
| `tags` | JSON | Topic tags for filtering |
| `org` | string nullable | Organisation scope (e.g. "core", "lthn", "ofm" — null = global) |
| `project` | string nullable | Repo/project scope (null = cross-project) |
| `indexed_at` | timestamp nullable | When Qdrant/ES indexing completed (null = pending async embed) |
| `confidence` | float | 0.0-1.0 |
| `supersedes_id` | UUID nullable | FK to older memory this replaces |
| `expires_at` | timestamp nullable | TTL for session-scoped context |

### Flow

```
brain_remember(content, tags, type)
  -> Store in MariaDB (brain_memories)
  -> Embed via Ollama (nomic-embed-text -> 768d vector)
  -> Upsert to Qdrant (point ID = MariaDB UUID)

brain_recall(query, filters)
  -> Embed query via Ollama
  -> Search Qdrant (cosine similarity, filtered by workspace + optional type/project/agent)
  -> Hydrate from MariaDB (full content + metadata)
  -> Return top-K results with similarity scores
```

### Memory Lifecycle

- **Supersession**: `supersedes_id` chains — new memory explicitly replaces old one.
- **TTL**: `expires_at` for session-scoped context that does not persist.
- **Confidence**: Agents set confidence; low-confidence memories rank lower in recall.
- **Soft delete**: `deleted_at` — memories are never hard deleted.

---

## 5. API Surface

Both implementations expose these capabilities but with different storage backends:

- **Go** operates on **local workspace state** — plans, sessions, and findings live in `.core/` filesystem and DuckDB. Go is the local agent runtime.
- **PHP** operates on **persistent database state** — MariaDB, Qdrant, Elasticsearch. PHP is the fleet coordination platform.
- **Sync** connects them: `POST /v1/agent/sync` pushes Go's local dispatch history/findings to PHP's persistent store. `GET /v1/agent/context` pulls fleet-wide intelligence back to Go.

Plans created locally by Go are workspace artifacts. Plans created via PHP are persistent. Cross-agent plan handoff requires syncing through the API. Go MCP tools operate on local plans; PHP REST endpoints operate on database plans.

### Brain (`/v1/brain/*`)

| Method | Endpoint | Action |
|--------|----------|--------|
| POST | `/v1/brain/remember` | Store knowledge |
| POST | `/v1/brain/recall` | Semantic search |
| DELETE | `/v1/brain/forget/{id}` | Remove memory |
| GET | `/v1/brain/list` | List memories |

### Plans (`/v1/plans/*`)

| Method | Endpoint | Action |
|--------|----------|--------|
| POST | `/v1/plans` | Create plan |
| GET | `/v1/plans` | List plans |
| GET | `/v1/plans/{id}` | Get plan |
| PATCH | `/v1/plans/{id}/status` | Update plan status |
| DELETE | `/v1/plans/{id}` | Archive plan |

### Sessions (`/v1/sessions/*`)

| Method | Endpoint | Action |
|--------|----------|--------|
| POST | `/v1/sessions` | Start session |
| GET | `/v1/sessions` | List sessions |
| GET | `/v1/sessions/{id}` | Get session |
| POST | `/v1/sessions/{id}/continue` | Resume session |
| POST | `/v1/sessions/{id}/end` | End session |

### Messages (`/v1/messages/*`)

| Method | Endpoint | Action |
|--------|----------|--------|
| POST | `/v1/messages/send` | Send message |
| GET | `/v1/messages/inbox` | Read inbox |
| GET | `/v1/messages/conversation/{agent}` | Get conversation thread |

### Issues, Sprints, Tasks, Phases

Standard CRUD patterns matching the domain model.

---

## 6. MCP Tools

Go exposes all tools via the core-agent MCP server binary. PHP exposes Brain, Plan, Session, and Message tools via the AgentToolRegistry. Dispatch, Workspace, and Forge tools are Go-only (PHP handles these via REST endpoints, not MCP tools).

### Brain Tools

| Tool Name | Maps To |
|-----------|---------|
| `brain_remember` | Store knowledge with embedding |
| `brain_recall` | Semantic search |
| `brain_forget` | Remove memory |
| `brain_list` | List memories |

### Dispatch Tools

| Tool Name | Maps To |
|-----------|---------|
| `agentic_dispatch` | Dispatch agent to workspace |
| `agentic_status` | Query workspace status |
| `agentic_scan` | Scan Forge for work |
| `agentic_watch` | Watch workspace output |
| `agentic_resume` | Resume agent |
| `agentic_review_queue` | List review queue |
| `agentic_dispatch_start` | Start dispatch service |
| `agentic_dispatch_shutdown` | Graceful shutdown (drain queue) |
| `agentic_dispatch_shutdown_now` | Immediate shutdown (kill running agents) |

### Workspace Tools

| Tool Name | Maps To |
|-----------|---------|
| `agentic_prep_workspace` | Prepare workspace |
| `agentic_create_epic` | Create epic |
| `agentic_create_pr` | Create pull request |
| `agentic_list_prs` | List pull requests |
| `agentic_mirror` | Mirror to remote |

### Plan Tools

| Tool Name | Maps To |
|-----------|---------|
| `agentic_plan_create` | Create plan |
| `agentic_plan_read` | Read plan |
| `agentic_plan_update` | Update plan |
| `agentic_plan_list` | List plans |
| `agentic_plan_delete` | Archive plan |

### Messaging Tools

| Tool Name | Maps To |
|-----------|---------|
| `agent_send` | Send message |
| `agent_inbox` | Read inbox |
| `agent_conversation` | Get conversation thread |

### Content Tools (PHP only)

| Tool Name | Maps To |
|-----------|---------|
| `content_generate` | Generate content from brief + prompt template |
| `content_batch` | Batch generation across services |
| `content_brief_create` | Create new product brief |

---

## 7. Completion Pipeline

When an agent completes, a handler chain fires:

```
AgentCompleted -> QA handler -> QAResult
QAResult{Passed} -> PR handler -> PRCreated
PRCreated -> Verify handler -> PRMerged | PRNeedsReview
AgentCompleted -> Ingest handler (findings -> issues)
AgentCompleted -> Poke handler (drain queue)
```

### Pipeline Steps

| Step | Action | Description |
|------|--------|-------------|
| 1 | QA | Run core/lint, capture ALL findings to workspace DuckDB |
| 2 | Auto-PR | Create pull request from passing output |
| 3 | Verify | Check CI status and review criteria |
| 4 (async) | Ingest | Extract findings and create Forge issues |
| 5 (async) | Poke | Drain the queue — dispatch next waiting task |
| 6 (async) | Commit | Workspace DuckDB → go-store journal (InfluxDB) |

Both implementations compose these as a Task (Go) or a Pipeline (PHP). The async steps run in parallel after Verify completes.

### QA with core/lint + go-store

The QA step captures EVERYTHING — the agent does not filter what it thinks is relevant. Raw findings go to the workspace DuckDB. The intelligence comes from analysis after, not during.

```go
// QA handler — runs lint, captures all findings to workspace store
func (s *QASubsystem) runQA(ctx context.Context, wsDir, repoDir string) QAResult {
    // Open workspace buffer for this dispatch cycle
    ws, err := s.store.NewWorkspace(core.Concat("qa-", core.PathBase(wsDir)))
    if err != nil {
        return QAResult{Error: core.E("qa.workspace", "create", err)}
    }

    // Run core/lint — capture every finding
    lintResult := s.core.Action("lint.run").Run(ctx, s.core, core.Options{
        "path":   repoDir,
        "output": "json",
    })
    var report lint.Report
    if r, ok := lintResult.Value.(lint.Report); ok {
        report = r
        for _, finding := range report.Findings {
            ws.Put("finding", map[string]any{
                "tool":     finding.Tool,
                "file":     finding.File,
                "line":     finding.Line,
                "severity": finding.Severity,
                "code":     finding.Code,
                "message":  finding.Message,
                "category": finding.Category,
            })
        }
        for _, tool := range report.Tools {
            ws.Put("tool_run", map[string]any{
                "name":     tool.Name,
                "status":   tool.Status,
                "duration": tool.Duration,
                "findings": tool.Findings,
            })
        }
    }

    // Run build
    buildResult := s.core.Action("process.run").Run(ctx, s.core, core.Options{
        "command": "go", "args": "build ./...", "dir": repoDir,
    })
    ws.Put("build", map[string]any{
        "passed": buildResult.OK,
        "output": buildResult.Value,
    })

    // Run tests
    testResult := s.core.Action("process.run").Run(ctx, s.core, core.Options{
        "command": "go", "args": "test ./... -count=1 -timeout 60s", "dir": repoDir,
    })
    ws.Put("test", map[string]any{
        "passed": testResult.OK,
        "output": testResult.Value,
    })

    // Commit the full cycle to journal — one entry per dispatch
    ws.Commit()

    // Return pass/fail based on lint errors + build + tests
    passed := buildResult.OK && testResult.OK
    return QAResult{
        Passed:   passed,
        Findings: len(report.Findings),
        Errors:   report.Summary.Errors,
    }
}
```

### Observability via Uptelligence

The journal tracks every dispatch cycle over time. Uptelligence analyses trends:

```
Query: "Which findings never get fixed?"
  → InfluxDB: findings that appear in 5+ consecutive cycles for the same repo
  → Result: gosec finding X in go-io has persisted for 12 cycles
  → Action: adjust CODEX template, update AX RFC, or change linter config

Query: "Did principle 6 reduce import violations?"
  → InfluxDB: count of 'banned_import' findings before and after RFC update
  → Result: 47 → 3 across 15 repos in 2 weeks
  → Proof: the methodology works, measured not assumed

Query: "Which repos spike errors after a dependency update?"
  → InfluxDB: build failures correlated with go.mod changes
  → Result: go-io fails after every core/go update
  → Action: pin version or fix the breaking change
```

No black box. Every warning is captured. Patterns emerge from the data, not from guessing.

### Post-Run Analysis (Poindexter)

Before `ws.Commit()`, the workspace DuckDB is analysed using Poindexter's multi-dimensional indexing. Each finding becomes a point in N-dimensional space — tool, severity, file, category, frequency. Poindexter's KD-tree clusters similar findings and cosine distance identifies patterns.

```go
// Analyse workspace before commit — extract insights from raw findings
func (s *QASubsystem) analyseWorkspace(ws *store.Workspace) DispatchReport {
    findings := ws.Query("SELECT tool, severity, file, category, COUNT(*) as n FROM entries WHERE kind='finding' GROUP BY tool, severity, file, category")

    // Build N-dimensional points from findings
    // Dimensions: tool_id, severity_score, file_hash, category_id, frequency
    var points []poindexter.Point
    for _, row := range findings.Value.([]map[string]any) {
        points = append(points, findingToPoint(row))
    }

    // Cluster similar findings
    tree := poindexter.BuildND(points, 5)
    clusters := tree.ClusterByDistance(0.15) // cosine distance threshold

    // Compare with previous journal entries to detect:
    // - New findings (not in previous cycles)
    // - Resolved findings (in previous, not in current)
    // - Persistent findings (in N+ consecutive cycles)
    previous := s.store.QueryJournal(core.Sprintf(
        `from(bucket: "core") |> range(start: -7d) |> filter(fn: (r) => r._measurement == "dispatch-%s")`,
        ws.Name(),
    ))

    return DispatchReport{
        Clusters:   clusters,
        New:        diffFindings(findings, previous, "new"),
        Resolved:   diffFindings(previous, findings, "resolved"),
        Persistent: persistentFindings(findings, previous, 5), // 5+ cycles
        Summary:    ws.Aggregate(),
    }
}

// DispatchReport is the analysis output before journal commit.
type DispatchReport struct {
    Clusters   []poindexter.Cluster   // grouped similar findings
    New        []map[string]any       // findings not seen before
    Resolved   []map[string]any       // findings that disappeared
    Persistent []map[string]any       // findings that won't go away
    Summary    map[string]any         // aggregated workspace state
}
```

The report is written to `.meta/report.json` in the workspace for human review. The aggregated summary goes to the journal via `ws.Commit()`. The raw DuckDB is then deleted — the intelligence survives in the report and the journal.

### Post-Completion Repo Sync

Workspace prep clones from the local repo, not Forge. If the local clone is stale, every dispatch builds on old code and produces duplicate changes. The sync must be event-driven, not polled.

**Event-driven sync (primary):**

```
QA passes → workspace pushes to Forge
  → IPC: WorkspacePushed{Repo, Branch, Org}
    → go-scm service handles event:
      → git fetch origin {branch} (in ~/Code/{org}/{repo})
      → git reset --hard origin/{branch}
    → local clone now matches Forge
    → next dispatch gets fresh code
```

The go-scm service listens for `WorkspacePushed` IPC messages and syncs the affected local clone. This closes the loop: workspace pushes to Forge, local clone pulls from Forge, next workspace clones from local.

**Background fetch (fallback):**

```
Every 5 minutes:
  → for each repo in agents.yaml (or scanned from workspace root):
    → git fetch origin (lightweight — refs only, no checkout)
```

The background fetch is a safety net for pushes from other agents (Charon, manual pushes). The event-driven sync handles all dispatch pipeline pushes.

| Trigger | Action | Scope |
|---------|--------|-------|
| `WorkspacePushed` IPC | `git fetch origin {branch} && git reset --hard origin/{branch}` | Single repo |
| Background (5 min) | `git fetch origin` | All registered repos |
| Manual (`core-agent repo/sync`) | `git fetch origin` + optional `--reset` | Specified repos |

---

## 8. IPC Messages

Typed messages for inter-service communication:

### Agent Lifecycle

| Message | Fields |
|---------|--------|
| `AgentStarted` | Agent, Repo, Workspace |
| `AgentCompleted` | Agent, Repo, Workspace, Status |

### Pipeline

| Message | Fields |
|---------|--------|
| `QAResult` | Workspace, Repo, Passed |
| `PRCreated` | Repo, Branch, PRURL, PRNum |
| `PRMerged` | Repo, PRURL, PRNum |
| `PRNeedsReview` | Repo, PRURL, PRNum, Reason |
| `WorkspacePushed` | Repo, Branch, Org |

### Queue

| Message | Fields |
|---------|--------|
| `QueueDrained` | Completed |
| `PokeQueue` | (empty) |

### Monitor

| Message | Fields |
|---------|--------|
| `HarvestComplete` | Repo, Branch, Files |
| `HarvestRejected` | Repo, Branch, Reason |
| `InboxMessage` | New, Total |

---

## 9. Fleet Mode

core-agent connects to the platform API for fleet-scale dispatch:

```
core-agent fleet --api=https://api.lthn.ai --agent-id=charon
```

### Connection

- AgentApiKey authentication. Bootstrap: `core login CODE` exchanges a 6-digit pairing code (generated at app.lthn.ai/device by a logged-in user) for an AgentApiKey. See lthn.ai RFC §11.7 Device Pairing. No OAuth needed — session auth on the web side, code exchange on the agent side.
- SSE connection for real-time job push
- Polling fallback for NAT'd nodes (`GET /v1/fleet/task/next`)
- Heartbeat and capability registration (`POST /v1/fleet/heartbeat`)

### Flow

1. Agent connects and registers capabilities
2. Platform pushes jobs via SSE (or agent polls)
3. Agent accepts job and dispatches locally
4. Agent reports result back to platform
5. Platform updates plan/session/issue state

This enables community onboarding — anyone running core-agent contributes compute.

---

## 10. Configuration

### agents.yaml

```yaml
version: 1
dispatch:
  default_agent: claude
  default_template: coding
  workspace_root: .core/workspace

# Per-pool concurrency (0 = unlimited)
concurrency:
  claude:
    total: 3
    opus: 1
    sonnet: 2
    haiku: 3
  gemini: 1
  codex: 2

# Rate limiting per pool
rates:
  claude:
    daily_limit: 50
    min_delay: 30
    sustained_delay: 60
    burst_window: 5
    burst_delay: 300
  codex:
    daily_limit: 0
    min_delay: 0
  codex-spark:
    min_delay: 10
    sustained_delay: 30

# Named agent identities
agents:
  cladius:
    host: local
    runner: claude
    roles: [dispatch, review, plan]
  charon:
    host: remote
    runner: claude
    roles: [worker, review]
```

### Codex Model Variants

Dispatch with `agent: codex:{model}`:

| Model | Use Case |
|-------|----------|
| `gpt-5.4` | Latest frontier, heavy tasks (default for `codex`) |
| `gpt-5.4-mini` | Moderate tasks |
| `gpt-5.3-codex` | Codex-optimised, code generation |
| `gpt-5.3-codex-spark` | Ultra-fast, AX sweeps and reviews |
| `gpt-5.2-codex` | Previous gen, stable |
| `gpt-5.2` | Professional work, long-running |
| `gpt-5.1-codex-max` | Deep reasoning |
| `gpt-5.1-codex-mini` | Cheap and fast |

### Queue Drain

When a dispatch completes or a slot frees up, the runner:
1. Checks concurrency limits (total + per-model)
2. Checks rate limits (daily, min_delay, burst window)
3. Pops next queued task matching an available pool
4. Spawns agent in sandboxed workspace
5. Emits `AgentStarted` -> runs -> emits `AgentCompleted`

---

## 11. Agent Identities

| Agent | Host | Runner | Roles | Description |
|-------|------|--------|-------|-------------|
| `cladius` | local (M3 Studio) | claude | dispatch, review, plan | Project leader, design sessions, orchestration |
| `charon` | remote (homelab) | claude | worker, review | Execution agent, bulk tasks, parallel work |
| `codex` | cloud | openai | worker | Code generation, sweeps, AX compliance |
| `clotho` | local | claude | review, qa | Quality gate, code review, test generation |

Agents communicate via `agent_send`/`agent_inbox` tools. Each agent has a unique `agent_id` used for brain memory attribution, session ownership, and message routing.

---

## 12. Content Generation Pipeline

The agentic module drives AI-powered content generation for the Host UK platform.

### Pipeline

```
Product Briefs (per service)
  -> Prompt Templates (system + user, versioned)
    -> AI Generation (Claude/Gemini via provider abstraction)
      -> Drafts (blog posts, help articles, social media)
        -> Quality Refinement (scoring, rewriting)
          -> Publication (CMS, social scheduler, help desk)
```

### Product Briefs

Each service has a brief that gives AI the product context:

| Brief | Product |
|-------|---------|
| `host-link.md` | LinkHost |
| `host-social.md` | SocialHost |
| `host-analytics.md` | AnalyticsHost |
| `host-trust.md` | TrustHost |
| `host-notify.md` | NotifyHost |

### Prompt Templates

Versioned prompt templates in categories:

| Category | Templates |
|----------|-----------|
| **Content** | blog-post, help-article, landing-page, social-media, quality-refinement |
| **Development** | architecture-review, code-review, debug-session, test-generation |
| **Visual** | infographic, logo-generation, social-graphics |
| **System** | dappcore-writer (brand voice) |

### Natural Progression SEO

Content changes create future revisions (scheduled posts with no date). When Googlebot visits a page with pending revisions, the system schedules publication 8-62 minutes later — making updates appear as natural content evolution rather than bulk changes.

### SEO Schema Generation

Structured data templates for generated content:
- Article (BlogPosting, TechArticle)
- FAQ (FAQPage)
- HowTo (step-by-step guides)

---

## 13. Session Lifecycle

```
StartSession(plan_id, agent) -> active session with context
  -> Agent works, appends to work_log
  -> ContinueSession(id, work) -> resume from last state
  -> EndSession(id, summary, handoff_notes) -> closed
  -> session_handoff: {summary, next_steps, blockers, context_for_next}
  -> session_replay: recover context from completed session
```

### Workspace State

Key-value store shared between sessions within a plan. When Agent A discovers something and stores it, Agent B reads it later from the same plan context. Types are enforced — values are not arbitrary strings.

---

## 14. Polyglot Mapping

| Go (core/go/agent) | PHP (core/php/agent) | Contract Capability |
|---------------------|----------------------|---------------------|
| `pkg/brain/*` | `Actions/Brain/*` | brain_remember/recall/forget |
| `pkg/brain/messaging.go` | `Actions/Messages/*` | Agent-to-agent messaging (send, inbox, conversation) |
| `pkg/agentic/plan.go` | `Actions/Plan/*` | Plan CRUD (via API) |
| `pkg/agentic/dispatch.go` | `Console/Commands/DispatchCommand` | Dispatch |
| `pkg/agentic/scan.go` | `Actions/Forge/ScanForWork` | Forge scan |
| `pkg/agentic/transport.go` | `Services/ForgejoService` | Forgejo API |
| `pkg/agentic/actions.go` | `Mcp/Tools/*` | MCP tool registration |
| `pkg/agentic/commands.go` | `Console/Commands/*` | CLI commands |
| `pkg/monitor/` | Admin UI (Livewire) | Monitoring and notifications |
| MCP tools | `Controllers/Api/*` | API surface |
| SQLite/file | MariaDB (Eloquent ORM) | Data layer |

**Key difference:** Go is the local MCP server binary (dispatch, workspace, brain). PHP is the web platform (REST API, admin UI, persistent storage, content generation).

---

## 15. State Persistence (go-store)

### 15.1 Overview

Agent state (workspace registry, queue, concurrency counts) persists to disk via go-store. On restart, state loads from the store — no ghost agents, no lost queue, no manual cleanup.

If go-store is not loaded as a service, agent falls back to in-memory state (current behaviour). The persistence is an upgrade, not a hard dependency.

### 15.2 State Files

```
.core/db.duckdb                              → top-level agent state
.core/workspace/{org}/{repo}/db.duckdb       → per-workspace dispatch state
```

### 15.3 Top-Level State (.core/db.duckdb)

| Group | Key Pattern | Value | Purpose |
|-------|------------|-------|---------|
| `queue` | `{repo}/{branch}` | JSON: task, agent, status, priority | Dispatch queue survives restart |
| `concurrency` | `{agent-type}` | JSON: running count, limit | No over-dispatch after restart |
| `registry` | `{org}/{repo}/{workspace}` | JSON: status, PID, agent, branch | No ghost agents |

```go
// On startup — restore state from store
// OnStartup restores state from go-store. store.New is used directly —
// agent owns its own store instance, it does not use the Core DI service registry for this.
func (s *Service) OnStartup(ctx context.Context) core.Result {
    st, err := store.New(".core/db.duckdb")
    if err != nil {
        return core.Result{Value: core.E("agent.startup", "state store", err), OK: false}
    }

    // Restore queue — values are JSON strings stored via store.Set
    for key, val := range st.AllSeq("queue") {
        var task QueuedTask
        core.JSONUnmarshalString(val, &task)
        s.queue.Enqueue(task)
    }

    // Restore registry — check PIDs, mark dead agents as failed
    for key, val := range st.AllSeq("registry") {
        var ws WorkspaceStatus
        core.JSONUnmarshalString(val, &ws)
        if ws.Status == "running" && !pidAlive(ws.PID) {
            ws.Status = "failed"
            ws.Question = "Agent process died during restart"
        }
        s.registry.Set(key, ws)
    }

    return core.Result{OK: true}
}
```

### 15.4 Per-Workspace State

Each workspace gets its own DuckDB for the dispatch cycle — accumulates events (started, findings, commits, QA results) and commits the full cycle to the journal on completion:

```go
// Dispatch creates a workspace buffer
//
//   ws, _ := st.NewWorkspace("core/go-io/dev")
//   ws.Put("started", map[string]any{"agent": "codex:gpt-5.4", "task": task})
//   ... agent runs ...
//   ws.Put("finding", map[string]any{"file": "service.go", "line": 42, "message": "..."})
//   ws.Put("completed", map[string]any{"status": "passed", "insertions": 231})
//   ws.Commit()  // → go-store handles journal write (InfluxDB if configured in store)
```

### 15.5 Automatic Cleanup + Stats Capture

No manual `workspace/clean` command needed. On cleanup, stats are written to the parent `.core/workspace/db.duckdb` BEFORE the workspace directory is deleted:

```
Workspace completes → Poindexter analysis → ws.Commit() → journal entry written
  → Write stats to .core/workspace/db.duckdb (parent):
    - dispatch duration, agent, model, repo, branch
    - findings count by severity, tool, category
    - build/test pass/fail
    - insertions/deletions
    - DispatchReport summary (clusters, new, resolved, persistent)
  → top-level registry entry updated to "completed"
  → workspace DuckDB file purged
  → workspace directory deleted

On startup: scan .core/workspace/ for orphaned workspace dirs
  → check parent db.duckdb registry — if "running" but PID dead → mark failed
  → if "completed" and workspace dir still exists → clean up
```

The parent `.core/workspace/db.duckdb` is the permanent record. Individual workspace dirs are disposable. "What happened in the last 50 dispatches?" is a query on the parent, not a scan of workspace dirs.

### 15.5.1 Branch Cleanup

After successful push or merge, delete the agent branch on Forge:

```go
// Clean up Forge branch after push
func (s *Service) cleanupBranch(ctx context.Context, repo, branch string) {
    s.core.Action("agentic.branch.delete").Run(ctx, s.core, core.Options{
        "repo":   repo,
        "branch": branch,
    })
}
```

Agent branches (`agent/*`) are ephemeral — they exist only during the dispatch cycle. Accumulation of stale branches pollutes the workspace prep and causes clone confusion.

### 15.5.2 Workspace Mount

The dispatch container mounts the workspace directory as the agent's home. The repo is at `repo/` within the workspace. Specs are baked into the Docker image at `~/spec/` (read-only, COPY at build time). The entrypoint handles auth symlinks and spec availability.

### 15.5.3 Apple Container Dispatch

On macOS 26+, agent dispatch uses Apple Containers instead of Docker. Apple Containers provide hardware VM isolation with sub-second startup — no Docker Desktop required, no cold-start penalty, and agents cannot escape the sandbox even with root.

The container runtime is auto-detected via go-container's `Detect()` function, which probes available runtimes in preference order: Apple Container, Docker, Podman. The first available runtime is used unless overridden in `agents.yaml` or per-dispatch options.

The container image is immutable — built by go-build's LinuxKit builder, not by the agent. The OS environment (toolchains, dependencies, linters) is enforced at build time. Agents work inside a known environment regardless of host configuration.

```go
// Dispatch an agent to an Apple Container workspace
//
//   agent.Dispatch(task, agent.WithRuntime(container.Apple),
//       agent.WithImage(build.LinuxKit("core-dev")),
//       agent.WithMount("~/Code/project", "/workspace"),
//       agent.WithGPU(true),  // Metal passthrough when available
//   )
func (s *Service) dispatchAppleContainer(ctx context.Context, task DispatchTask) core.Result {
    // Detect runtime — prefers Apple → Docker → Podman
    rt := s.Core().Action("container.detect").Run(ctx, s.Core(), core.Options{})
    runtime := rt.Value.(string) // "apple", "docker", "podman"

    // Resolve immutable image — built by go-build LinuxKit
    image := s.Core().Action("build.linuxkit.resolve").Run(ctx, s.Core(), core.Options{
        "base": task.Image, // "core-dev", "core-ml", "core-minimal"
    })

    return s.Core().Action("container.run").Run(ctx, s.Core(), core.Options{
        "runtime": runtime,
        "image":   image.Value.(string),
        "mount":   core.Concat(task.WorkspaceDir, ":/workspace"),
        "gpu":     task.GPU,
        "env":     task.Env,
        "command": task.Command,
    })
}
```

**Runtime behaviour:**

| Property | Apple Container | Docker | Podman |
|----------|----------------|--------|--------|
| Isolation | Hardware VM (Virtualisation.framework) | Namespace/cgroup | Namespace/cgroup |
| Startup | Sub-second | 2-5 seconds (cold) | 2-5 seconds (cold) |
| GPU | Metal passthrough (roadmap) | NVIDIA only | NVIDIA only |
| Root escape | Impossible (VM boundary) | Possible (misconfigured) | Possible (rootless mitigates) |
| macOS native | Yes | Requires Docker Desktop | Requires Podman Machine |

**Fallback chain:** If Apple Containers are unavailable (macOS < 26, Linux host, CI environment), dispatch falls back to Docker automatically. The agent code is runtime-agnostic — the same `container.run` action handles all three runtimes.

**GPU passthrough:** Metal GPU passthrough is on Apple's roadmap. When available, `agent.WithGPU(true)` enables it — go-mlx works inside the container for local inference during agent tasks. Until then, `WithGPU(true)` is a no-op on Apple Containers and enables NVIDIA passthrough on Docker.

**Configuration:**

```yaml
# agents.yaml — runtime preference override
dispatch:
  runtime: auto          # auto | apple | docker | podman
  image: core-dev        # default LinuxKit image
  gpu: false             # Metal passthrough (when available)
```

### 15.6 Graceful Degradation

```go
// If go-store is loaded, use it. If not, fall back to in-memory.
func (s *Service) stateStore() *store.Store {
    if s.store != nil {
        return s.store
    }
    return nil  // callers check nil and use in-memory maps
}
```

Agent checks `s.store != nil` before any store call. If go-store is not initialised (New fails or is skipped), all state falls back to in-memory maps. No IPC dependency, no crashes, no hard dependency.

### 15.7 CLI Test Validation (AX-10)

Before swapping the core-agent binary, the CLI tests validate state persistence:

```
tests/cli/core/agent/
├── dispatch/
│   ├── Taskfile.yaml      ← test dispatch + restart + queue survives
│   └── fixtures/
├── status/
│   ├── Taskfile.yaml      ← test status after restart shows correct state
│   └── fixtures/
├── restart/
│   ├── Taskfile.yaml      ← test: dispatch → kill → restart → no ghost agents
│   └── fixtures/
└── clean/
    ├── Taskfile.yaml      ← test: completed workspaces auto-cleaned
    └── fixtures/
```

Build binary → run tests → pass? swap. Fail? keep backup. No scratch card.

---

## 16. Remote State Sync (lthn.ai)

### 16.1 Overview

Agents authenticated with api.lthn.ai can sync local state to the platform. Local `.core/db.duckdb` state pushes to core/php/agent endpoints, which update OpenBrain embeddings and managed workflow state. Any authed agent in the fleet gets shared context.

```
Local agent (.core/db.duckdb)
  → auth: api.lthn.ai (AgentApiKey)
    → POST /v1/agent/sync (dispatches[] — see DispatchHistoryItem below)
      → core/php/agent receives state

DispatchHistoryItem payload shape (Go produces, PHP consumes):
  { id (UUID, generated at dispatch time), repo, branch, agent_model, task, template, status, started_at, completed_at,
    findings: [{tool, severity, file, category, message}],
    changes: {files_changed, insertions, deletions},
    report: {clusters_count, new_count, resolved_count, persistent_count},
    synced: false }

        → OpenBrain: embed findings as BrainMemory records
        → WorkspaceState: update managed workflow progress
        → Notify: alert subscribers of new findings
  → GET /v1/agent/context (pull shared state from fleet)
    → Other agents' findings, resolved patterns, fleet-wide trends
```

### 16.2 Sync Actions

```go
func (s *Service) OnStartup(ctx context.Context) core.Result {
    c := s.Core()

    c.Action("agent.sync.push", s.handleSyncPush)
    c.Action("agent.sync.pull", s.handleSyncPull)

    return core.Result{OK: true}
}
```

| Action | Input | Effect |
|--------|-------|--------|
| `agent.sync.push` | (none — reads from local db.duckdb) | Push dispatch history + findings to api.lthn.ai |
| `agent.sync.pull` | (none — writes to local db.duckdb) | Pull fleet-wide context from api.lthn.ai |

### 16.3 Push Payload

```go
// SyncPush reads completed dispatch cycles from .core/db.duckdb
// and POSTs them to api.lthn.ai/v1/agent/sync
func (s *Service) handleSyncPush(ctx context.Context, opts core.Options) core.Result {
    st := s.stateStore()
    if st == nil {
        return core.Result{OK: false, Value: core.E("agent.sync.push", "no store", nil)}
    }

    // Collect unsync'd dispatch records
    var payload []map[string]any
    for key, val := range st.AllSeq("dispatch_history") {
        var record map[string]any
        core.JSONUnmarshalString(val, &record)
        if synced, _ := record["synced"].(bool); !synced {
            payload = append(payload, record)
        }
    }

    if len(payload) == 0 {
        return core.Result{OK: true} // nothing to sync
    }

    // POST to lthn.ai
    result := s.Core().Action("api.post").Run(ctx, s.Core(), core.Options{
        "url":  core.Concat(s.apiURL, "/v1/agent/sync"),
        "body": core.JSONMarshalString(payload),
        "auth": s.apiKey,
    })

    // Mark records as synced
    if result.OK {
        for _, record := range payload {
            record["synced"] = true
            st.Set("dispatch_history", record["id"].(string), core.JSONMarshalString(record))
        }
    }

    return result
}
```

### 16.4 Pull Context

```go
// SyncPull fetches fleet-wide context from api.lthn.ai/v1/agent/context
// and merges it into the local store for use during dispatch
func (s *Service) handleSyncPull(ctx context.Context, opts core.Options) core.Result {
    result := s.Core().Action("api.get").Run(ctx, s.Core(), core.Options{
        "url":  core.Concat(s.apiURL, "/v1/agent/context"),
        "auth": s.apiKey,
    })

    if !result.OK {
        return result
    }

    // Merge fleet context into local store
    var context []map[string]any
    core.JSONUnmarshalString(result.Value.(string), &context)

    st := s.stateStore()
    for _, entry := range context {
        if id, ok := entry["id"].(string); ok {
            st.Set("fleet_context", id, core.JSONMarshalString(entry))
        }
    }

    return core.Result{OK: true}
}
```

### 16.5 Offline Queue

When api.lthn.ai is unreachable, results queue in `.core/db.duckdb`:

```go
// Queue structure in go-store
// Group: "sync_queue", Key: timestamp-based ID, Value: JSON payload
st.Set("sync_queue", core.Sprintf("sync-%d", time.Now().UnixMilli()), payload)

// Flush on reconnect — oldest first
for key, val := range st.AllSeq("sync_queue") {
    result := s.Core().Action("api.post").Run(ctx, s.Core(), core.Options{
        "url":  core.Concat(s.apiURL, "/v1/agent/sync"),
        "body": val,
        "auth": s.apiKey,
    })
    if result.OK {
        st.Delete("sync_queue", key)
    } else {
        break // stop on first failure, retry next cycle
    }
}
```

Backoff schedule: 1s → 5s → 15s → 60s → 5min (max). Queue persists across restarts in db.duckdb. Flush order: heartbeat first, then task completions (oldest first), then dispatch history.

### 16.6 Graceful Degradation

No API key = no sync. The agent works fully offline. Sync is additive — it enriches context but is never required. If api.lthn.ai is unreachable, the push queue accumulates in db.duckdb and flushes on next successful connection.

### 16.6 PHP Endpoints (core/php/agent)

The PHP side receives sync pushes and serves context pulls:

| Endpoint | Method | Purpose |
|----------|--------|---------|
| `/v1/agent/sync` | POST | Receive dispatch history, findings. Write to BrainMemory + WorkspaceState |
| `/v1/agent/context` | GET | Return fleet-wide findings, resolved patterns, persistent issues |
| `/v1/agent/status` | GET | Return agent's own sync status, last push time |

These endpoints map to existing core/php/agent Actions:
- `PushDispatchHistory` — receives push, creates BrainMemory records with embeddings
- `GetFleetContext` — queries BrainMemory for findings across all agents
- `GetAgentStatus` — returns sync metadata

See `code/core/php/agent/RFC.md` § "API Endpoints" and § "OpenBrain" for the PHP implementation.

---

## 17. Reference Material

| Resource | Location |
|----------|----------|
| Go implementation spec | `code/core/go/agent/RFC.md` |
| PHP implementation spec | `code/core/php/agent/RFC.md` |
| Core framework spec | `code/core/go/RFC.md` |
| Process primitives | `code/core/go/process/RFC.md` |
| Store (state persistence) | `code/core/go/store/RFC.md` |
| Poindexter (spatial analysis) | `code/snider/poindexter/RFC.md` |
| Lint (QA gate) | `code/core/lint/RFC.md` |
| MCP spec | `code/core/mcp/RFC.md` |
| RAG RFC | `code/core/go/rag/RFC.md` |

---

## Changelog

- 2026-04-08: Added §15.5.3 Apple Container Dispatch — native macOS 26 hardware VM isolation, auto-detected runtime fallback chain (Apple → Docker → Podman), immutable LinuxKit images from go-build, Metal GPU passthrough (roadmap).
- 2026-03-29: Restructured as language-agnostic contract. Go-specific code moved to `code/core/go/agent/RFC.md`. PHP-specific code stays in `code/core/php/agent/RFC.md`. Polyglot mapping, OpenBrain architecture, and completion pipeline consolidated here.
- 2026-03-26: WIP — net/http consolidated to transport.go.
- 2026-03-25: Initial spec — written with full core/go v0.8.0 domain context.
