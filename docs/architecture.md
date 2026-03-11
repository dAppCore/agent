---
title: Architecture
description: Internal architecture of core/agent — task lifecycle, dispatch pipeline, agent loop, orchestration, and the PHP backend.
---

# Architecture

Core Agent spans two runtimes (Go and PHP) that collaborate through a REST API. The Go side handles agent-side execution, CLI commands, and the autonomous agent loop. The PHP side provides the backend API, persistent storage, multi-provider AI services, and the admin panel.

```
                    Forgejo
                      |
             [ForgejoSource polls]
                      |
                      v
    +-- jobrunner Poller --+      +-- PHP Backend --+
    | ForgejoSource        |      | AgentApiController|
    | DispatchHandler  ----|----->| /v1/plans         |
    | CompletionHandler    |      | /v1/sessions      |
    | ResolveThreadsHandler|      | /v1/plans/*/phases|
    +----------------------+      +---------+---------+
                                            |
                                    [database models]
                                    AgentPlan, AgentPhase,
                                    AgentSession, BrainMemory
```


## Go: Task Lifecycle (`pkg/lifecycle/`)

The lifecycle package is the core domain layer. It defines the data types and orchestration logic for task management.

### Key Types

**Task** represents a unit of work:

```go
type Task struct {
    ID           string       `json:"id"`
    Title        string       `json:"title"`
    Description  string       `json:"description"`
    Priority     TaskPriority `json:"priority"`     // critical, high, medium, low
    Status       TaskStatus   `json:"status"`        // pending, in_progress, completed, blocked, failed
    Labels       []string     `json:"labels,omitempty"`
    Files        []string     `json:"files,omitempty"`
    Dependencies []string     `json:"dependencies,omitempty"`
    MaxRetries   int          `json:"max_retries,omitempty"`
    RetryCount   int          `json:"retry_count,omitempty"`
    // ...timestamps, claimed_by, etc.
}
```

**AgentInfo** describes a registered agent:

```go
type AgentInfo struct {
    ID            string      `json:"id"`
    Name          string      `json:"name"`
    Capabilities  []string    `json:"capabilities,omitempty"`
    Status        AgentStatus `json:"status"`  // available, busy, offline
    LastHeartbeat time.Time   `json:"last_heartbeat"`
    CurrentLoad   int         `json:"current_load"`
    MaxLoad       int         `json:"max_load"`
}
```

### Agent Registry

The `AgentRegistry` interface tracks agent availability with heartbeats and reaping:

```go
type AgentRegistry interface {
    Register(agent AgentInfo) error
    Deregister(id string) error
    Get(id string) (AgentInfo, error)
    List() []AgentInfo
    All() iter.Seq[AgentInfo]
    Heartbeat(id string) error
    Reap(ttl time.Duration) []string
}
```

Three backends are provided:
- `MemoryRegistry` -- in-process, mutex-guarded, copy-on-read
- `SQLiteRegistry` -- persistent, single-file database
- `RedisRegistry` -- distributed, suitable for multi-node deployments

Backend selection is driven by `RegistryConfig`:

```go
registry, err := NewAgentRegistryFromConfig(RegistryConfig{
    RegistryBackend: "sqlite",  // "memory", "sqlite", or "redis"
    RegistryPath:    "/path/to/registry.db",
})
```

### Task Router

The `TaskRouter` interface selects agents for tasks. The `DefaultRouter` implements capability matching and load-based scoring:

1. **Filter** -- only agents that are `Available` (or `Busy` with capacity) and possess all required capabilities (matched via task labels).
2. **Critical tasks** -- pick the least-loaded agent directly.
3. **Other tasks** -- score by availability ratio (`1.0 - currentLoad/maxLoad`) and pick the highest-scored agent. Ties are broken alphabetically for determinism.

### Allowance System

The allowance system enforces quota limits to prevent runaway costs. It operates at two levels:

**Per-agent quotas** (`AgentAllowance`):
- Daily token limit
- Daily job limit
- Concurrent job limit
- Maximum job duration
- Model allowlist

**Per-model quotas** (`ModelQuota`):
- Daily token budget (global across all agents)
- Hourly rate limit (reserved, not yet enforced)
- Cost ceiling (reserved, not yet enforced)

The `AllowanceService` provides:
- `Check(agentID, model)` -- pre-dispatch gate that returns `QuotaCheckResult`
- `RecordUsage(report)` -- updates counters based on `QuotaEvent` (started/completed/failed/cancelled)

Quota recovery: failed jobs return 50% of tokens; cancelled jobs return 100%.

Three storage backends mirror the registry: `MemoryStore`, `SQLiteStore`, `RedisStore`.

### Dispatcher

The `Dispatcher` orchestrates the full dispatch cycle:

```
1. List available agents  (AgentRegistry)
2. Route task to agent    (TaskRouter)
3. Check allowance        (AllowanceService)
4. Claim task via API     (Client)
5. Record usage           (AllowanceService)
6. Emit events            (EventEmitter)
```

`DispatchLoop` polls for pending tasks at a configurable interval, sorts by priority (critical first, oldest first as tie-breaker), and dispatches each one. Failed dispatches are retried with exponential backoff (5s, 10s, 20s, ...). Tasks exceeding their retry limit are dead-lettered with `StatusFailed`.

### Event System

Lifecycle events are published through the `EventEmitter` interface:

| Event | When |
|-------|------|
| `task_dispatched` | Task successfully routed and claimed |
| `task_claimed` | API claim succeeded |
| `dispatch_failed_no_agent` | No eligible agent available |
| `dispatch_failed_quota` | Agent quota exceeded |
| `task_dead_lettered` | Task exceeded retry limit |
| `quota_warning` | Agent at 80%+ usage |
| `quota_exceeded` | Agent over quota |
| `usage_recorded` | Usage counters updated |

Two emitter implementations:
- `ChannelEmitter` -- buffered channel, drops events when full (non-blocking)
- `MultiEmitter` -- fans out to multiple emitters

### API Client

`Client` communicates with the PHP backend over HTTP:

```go
client := NewClient("https://api.lthn.sh", "your-token")
client.AgentID = "cladius"

tasks, _ := client.ListTasks(ctx, ListOptions{Status: StatusPending})
task, _ := client.ClaimTask(ctx, taskID)
_ = client.CompleteTask(ctx, taskID, TaskResult{Success: true})
```

Additional endpoints for plans, sessions, phases, and brain (OpenBrain) are available.

### Context Gathering

`BuildTaskContext` assembles rich context for AI consumption:

1. Reads files explicitly mentioned in the task
2. Runs `git status` and `git log`
3. Searches for related code using keyword extraction + `git grep`
4. Formats everything into a markdown document via `FormatContext()`

### Service (Core DI Integration)

The `Service` struct integrates with the Core DI container. It registers task handlers for `TaskCommit` and `TaskPrompt` messages, executing Claude via subprocess:

```go
core.New(
    core.WithService(lifecycle.NewService(lifecycle.ServiceOptions{
        DefaultTools: []string{"Bash", "Read", "Glob", "Grep"},
        AllowEdit:    false,
    })),
)
```

### Embedded Prompts

Prompt templates are embedded at compile time from `prompts/*.md` and accessed via `Prompt(name)`.


## Go: Agent Loop (`pkg/loop/`)

The loop package implements an autonomous agent loop that drives any `inference.TextModel`:

```go
engine := loop.New(
    loop.WithModel(myTextModel),
    loop.WithTools(myTools...),
    loop.WithMaxTurns(10),
)

result, err := engine.Run(ctx, "Fix the failing test in pkg/foo")
```

### How It Works

1. Build a system prompt describing available tools
2. Send the user message to the model
3. Parse the response for `\`\`\`tool` fenced blocks
4. Execute matched tool handlers
5. Append tool results to the conversation history
6. Loop until the model responds without tool blocks, or `maxTurns` is reached

### Tool Definition

```go
loop.Tool{
    Name:        "read_file",
    Description: "Read a file from disk",
    Parameters:  map[string]any{"type": "object", ...},
    Handler: func(ctx context.Context, args map[string]any) (string, error) {
        path := args["path"].(string)
        return os.ReadFile(path)
    },
}
```

### Built-in Tool Adapters

- `LoadMCPTools(svc)` -- converts go-ai MCP tools into loop tools
- `EaaSTools(baseURL)` -- wraps the EaaS scoring API (score, imprint, atlas similar)


## Go: Job Runner (`pkg/jobrunner/`)

The jobrunner implements a poll-dispatch engine for CI/CD-style agent automation.

### Core Interfaces

```go
type JobSource interface {
    Name() string
    Poll(ctx context.Context) ([]*PipelineSignal, error)
    Report(ctx context.Context, result *ActionResult) error
}

type JobHandler interface {
    Name() string
    Match(signal *PipelineSignal) bool
    Execute(ctx context.Context, signal *PipelineSignal) (*ActionResult, error)
}
```

### Poller

The `Poller` ties sources and handlers together. On each cycle it:

1. Polls all sources for `PipelineSignal` values
2. Finds the first matching handler for each signal
3. Executes the handler (or logs in dry-run mode)
4. Records results in the `Journal` (JSONL audit log)
5. Reports back to the source

### Forgejo Source (`forgejo/`)

Polls Forgejo for epic issues (issues labelled `epic`), parses their body for linked child issues, and checks each child for a linked PR. Produces signals for:

- Children with PRs (includes PR state, check status, merge status, review threads)
- Children without PRs but with agent assignees (`NeedsCoding: true`)

### Handlers (`handlers/`)

| Handler | Matches | Action |
|---------|---------|--------|
| `DispatchHandler` | `NeedsCoding` + known agent assignee | Creates ticket JSON, transfers via SSH to agent queue |
| `CompletionHandler` | Agent completion signals | Updates Forgejo issue labels, ticks parent epic |
| `EnableAutoMergeHandler` | All checks passing, no unresolved threads | Enables auto-merge on the PR |
| `PublishDraftHandler` | Draft PRs with passing checks | Marks the PR as ready for review |
| `ResolveThreadsHandler` | PRs with unresolved threads | Resolves outdated review threads |
| `SendFixCommandHandler` | PRs with failing checks | Comments with fix instructions |
| `TickParentHandler` | Merged PRs | Checks off the child in the parent epic |

### Journal

The `Journal` writes date-partitioned JSONL files to `{baseDir}/{owner}/{repo}/{date}.jsonl`. Path components are sanitised to prevent traversal attacks.


## Go: Orchestrator (`pkg/orchestrator/`)

### Clotho Protocol

The orchestrator implements the "Clotho Protocol" for dual-run verification. When enabled, a task is executed twice with different models and the outputs are compared:

```go
spinner := orchestrator.NewSpinner(clothoConfig, agents)
mode := spinner.DeterminePlan(signal, agentName)
// mode is either ModeStandard or ModeDual
```

Dual-run is triggered when:
- The global strategy is `clotho-verified`
- The agent has `dual_run: true` in its config
- The repository is deemed critical (name is "core" or contains "security")

### Agent Configuration

```yaml
agentci:
  agents:
    cladius:
      host: user@192.168.1.100
      queue_dir: /home/claude/ai-work/queue
      forgejo_user: virgil
      model: sonnet
      runner: claude          # claude, codex, or gemini
      dual_run: false
      active: true
  clotho:
    strategy: direct          # direct or clotho-verified
    validation_threshold: 0.85
```

### Security

- `SanitizePath` -- validates filenames against `^[a-zA-Z0-9\-\_\.]+$` and rejects traversal
- `EscapeShellArg` -- single-quote wrapping for safe shell insertion
- `SecureSSHCommandContext` -- strict host key checking, batch mode, 10-second connect timeout
- `MaskToken` -- redacts tokens for safe logging


## Go: Dispatch (`cmd/dispatch/`)

The dispatch command runs **on the agent machine** and processes work from the PHP API:

### `core ai dispatch watch`

1. Connects to the PHP agentic API (`/v1/health` ping)
2. Lists active plans (`/v1/plans?status=active`)
3. Finds the first workable phase (in-progress or pending with `can_start`)
4. Starts a session via the API
5. Clones/updates the repository
6. Builds a prompt from the phase description
7. Invokes the runner (`claude`, `codex`, or `gemini`)
8. Reports success/failure back to the API and Forgejo

**Rate limiting**: if an agent exits in under 30 seconds (fast failure), the poller backs off exponentially (2x, 4x, 8x the base interval, capped at 8x).

### `core ai dispatch run`

Processes a single ticket from the local file queue (`~/ai-work/queue/ticket-*.json`). Uses file-based locking to prevent concurrent execution.


## Go: Workspace (`cmd/workspace/`)

### Task Workspaces

Each task gets an isolated workspace at `.core/workspace/p{epic}/i{issue}/` containing git worktrees:

```
.core/workspace/
  p42/
    i123/
      core-php/        # git worktree on branch issue/123
      core-tenant/     # git worktree on branch issue/123
      agents/
        claude-opus/implementor/
          memory.md
          artifacts/
```

Safety checks prevent removal of workspaces with uncommitted changes or unpushed branches.

### Agent Context

Agents get persistent directories within task workspaces. Each agent has a `memory.md` file that persists across invocations, allowing QA agents to accumulate findings and implementors to record decisions.


## Go: MCP Server (`cmd/mcp/`)

A standalone MCP server (stdio transport via mcp-go) exposing four tools:

| Tool | Purpose |
|------|---------|
| `marketplace_list` | Lists available Claude Code plugins from `marketplace.json` |
| `marketplace_plugin_info` | Returns metadata, commands, and skills for a plugin |
| `core_cli` | Runs approved `core` CLI commands (dev, go, php, build only) |
| `ethics_check` | Returns the Axioms of Life ethics modal and kernel |


## PHP: Backend API

### Service Provider (`Boot.php`)

The module registers via Laravel's event-driven lifecycle:

| Event | Handler | Purpose |
|-------|---------|---------|
| `ApiRoutesRegistering` | `onApiRoutes` | REST API endpoints at `/v1/*` |
| `AdminPanelBooting` | `onAdminPanel` | Livewire admin components |
| `ConsoleBooting` | `onConsole` | Artisan commands |
| `McpToolsRegistering` | `onMcpTools` | Brain MCP tools |

Scheduled commands:
- `agentic:plan-cleanup` -- daily plan retention
- `agentic:scan` -- every 5 minutes (Forgejo pipeline scan)
- `agentic:dispatch` -- every 2 minutes (agent dispatch)
- `agentic:pr-manage` -- every 5 minutes (PR lifecycle management)

### REST API Routes

All authenticated routes use `AgentApiAuth` middleware with Bearer tokens and scope-based permissions.

**Plans** (`/v1/plans`):
- `GET /v1/plans` -- list plans (filterable by status)
- `GET /v1/plans/{slug}` -- get plan with phases
- `POST /v1/plans` -- create plan
- `PATCH /v1/plans/{slug}` -- update plan
- `DELETE /v1/plans/{slug}` -- archive plan

**Phases** (`/v1/plans/{slug}/phases/{phase}`):
- `GET` -- get phase details
- `PATCH` -- update phase status
- `POST .../checkpoint` -- add checkpoint
- `PATCH .../tasks/{idx}` -- update task
- `POST .../tasks/{idx}/toggle` -- toggle task completion

**Sessions** (`/v1/sessions`):
- `GET /v1/sessions` -- list sessions
- `GET /v1/sessions/{id}` -- get session
- `POST /v1/sessions` -- start session
- `POST /v1/sessions/{id}/end` -- end session
- `POST /v1/sessions/{id}/continue` -- continue session

### Data Model

**AgentPlan** -- a structured work plan with phases, multi-tenant via `BelongsToWorkspace`:
- Status: draft -> active -> completed/archived
- Phases: ordered list of `AgentPhase` records
- Sessions: linked `AgentSession` records
- State: key-value `WorkspaceState` records

**AgentSession** -- tracks an agent's work session for handoff:
- Status: active -> paused -> completed/failed
- Work log: timestamped entries (info, warning, error, checkpoint, decision)
- Artifacts: files created/modified/deleted
- Handoff notes: summary, next steps, blockers, context for next agent
- Replay: `createReplaySession()` spawns a continuation session with inherited context

**BrainMemory** -- persistent knowledge stored in both MariaDB and Qdrant:
- Types: fact, decision, pattern, context, procedure
- Semantic search via Ollama embeddings + Qdrant vector similarity
- Supersession: new memories can replace old ones (soft delete + vector removal)

### AI Provider Management (`AgenticManager`)

Three providers are registered at boot:

| Provider | Service | Default Model |
|----------|---------|---------------|
| Claude | `ClaudeService` | `claude-sonnet-4-20250514` |
| Gemini | `GeminiService` | `gemini-2.0-flash` |
| OpenAI | `OpenAIService` | `gpt-4o-mini` |

Each implements `AgenticProviderInterface`. Missing API keys are logged as warnings at boot time.

### BrainService (OpenBrain)

The `BrainService` provides semantic memory using Ollama for embeddings and Qdrant for vector storage:

```
remember() -> embed(content) -> DB::transaction {
    BrainMemory::create() + qdrantUpsert()
    if supersedes_id: soft-delete old + qdrantDelete()
}

recall(query) -> embed(query) -> qdrantSearch() -> BrainMemory::whereIn(ids)
```

Default embedding model: `embeddinggemma` (768-dimensional vectors, cosine distance).


## Data Flow: End-to-End Dispatch

1. **PHP** `agentic:scan` scans Forgejo for issues labelled `agent-ready`
2. **PHP** `agentic:dispatch` creates plans with phases from issues
3. **Go** `core ai dispatch watch` polls `GET /v1/plans?status=active`
4. **Go** finds first workable phase, starts a session via `POST /v1/sessions`
5. **Go** clones the repository, builds a prompt, invokes the runner
6. **Runner** (Claude/Codex/Gemini) makes changes, commits, pushes
7. **Go** reports phase status via `PATCH /v1/plans/{slug}/phases/{phase}`
8. **Go** ends the session via `POST /v1/sessions/{id}/end`
9. **Go** comments on the Forgejo issue with the result
