# core/go/agent RFC — Go Agent Implementation

> The Go implementation of the agent system — dispatch, workspace management, MCP server.
> Implements `code/core/agent/RFC.md` contract in Go.
> An agent should be able to implement the Go agent from this document alone.

**Module:** `dappco.re/go/agent`
**Binary:** `~/.local/bin/core-agent`
**Depends on:** core/go v0.8.0, go-process v0.8.0
**Sub-specs:** [Models](RFC.models.md) | [Commands](RFC.commands.md)

---

## 1. Overview

core/go/agent is the local MCP server binary that dispatches AI agents, manages sandboxed workspaces, provides semantic memory (OpenBrain), and runs the completion pipeline. It composes core/go primitives (ServiceRuntime, Actions, Tasks, IPC, Process) into a single binary: `core-agent`.

The cross-cutting contract lives in `code/core/agent/RFC.md`. This document covers Go-specific patterns: service registration, named actions, process execution, status management, monitoring, MCP tools, runner service, dispatch routing, and quality gates.

---

## 2. Service Registration

All services use `ServiceRuntime[T]` — no raw `core *core.Core` fields.

```go
func Register(c *core.Core) core.Result {
    prep := NewPrep()
    prep.ServiceRuntime = core.NewServiceRuntime(c, AgentOptions{})

    cfg := prep.loadAgentsConfig()
    c.Config().Set("agents.concurrency", cfg.Concurrency)
    c.Config().Set("agents.rates", cfg.Rates)

    RegisterHandlers(c, prep)
    return core.Result{Value: prep, OK: true}
}

// In main:
c := core.New(
    core.WithService(process.Register),
    core.WithService(agentic.Register),
    core.WithService(brain.Register),
    core.WithService(monitor.Register),
    core.WithService(mcp.Register),
)
c.Run()
```

All subsystems embed `*core.ServiceRuntime[T]`:

```go
// pkg/agentic/ — PrepSubsystem
type PrepSubsystem struct {
    *core.ServiceRuntime[AgentOptions]
}

// pkg/brain/ — BrainService
type BrainService struct {
    *core.ServiceRuntime[BrainOptions]
}

// pkg/monitor/ — Monitor
type Monitor struct {
    *core.ServiceRuntime[MonitorOptions]
}

// pkg/setup/ — Setup Service
type Service struct {
    *core.ServiceRuntime[SetupOptions]
}
```

---

## 3. Named Actions

All capabilities registered as named Actions during OnStartup. Inspectable, composable, gatable by Entitlements.

```go
func (s *PrepSubsystem) OnStartup(ctx context.Context) core.Result {
    c := s.Core()

    // Dispatch & workspace
    c.Action("agentic.dispatch", s.handleDispatch)
    c.Action("agentic.prep", s.handlePrep)
    c.Action("agentic.status", s.handleStatus)
    c.Action("agentic.resume", s.handleResume)
    c.Action("agentic.scan", s.handleScan)
    c.Action("agentic.watch", s.handleWatch)

    // Pipeline
    c.Action("agentic.qa", s.handleQA)
    c.Action("agentic.auto-pr", s.handleAutoPR)
    c.Action("agentic.verify", s.handleVerify)
    c.Action("agentic.ingest", s.handleIngest)
    c.Action("agentic.poke", s.handlePoke)
    c.Action("agentic.mirror", s.handleMirror)

    // Forge
    c.Action("agentic.issue.get", s.handleIssueGet)
    c.Action("agentic.issue.list", s.handleIssueList)
    c.Action("agentic.issue.create", s.handleIssueCreate)
    c.Action("agentic.pr.get", s.handlePRGet)
    c.Action("agentic.pr.list", s.handlePRList)
    c.Action("agentic.pr.merge", s.handlePRMerge)

    // Review & Epic
    c.Action("agentic.review-queue", s.handleReviewQueue)
    c.Action("agentic.epic", s.handleEpic)

    // Completion pipeline — Task composition
    c.Task("agent.completion", core.Task{
        Description: "QA -> PR -> Verify -> Merge",
        Steps: []core.Step{
            {Action: "agentic.qa"},
            {Action: "agentic.auto-pr"},
            {Action: "agentic.verify"},
            {Action: "agentic.ingest", Async: true},
            {Action: "agentic.poke", Async: true},
        },
    })

    s.StartRunner()
    s.registerCommands(ctx)
    s.registerWorkspaceCommands()
    s.registerForgeCommands()
    return core.Result{OK: true}
}
```

### Entitlement Gating

Actions are gated by `c.Entitled()` — checked automatically in `Action.Run()`:

```go
func (s *PrepSubsystem) handleDispatch(ctx context.Context, opts core.Options) core.Result {
    e := s.Core().Entitled("agentic.concurrency", 1)
    if !e.Allowed {
        return core.Result{Value: core.E("dispatch", e.Reason, nil), OK: false}
    }
    // ... dispatch agent ...
    s.Core().RecordUsage("agentic.dispatch")
    return core.Result{OK: true}
}
```

### Remote Dispatch

Transparent local/remote via `host:action` syntax:

```go
r := c.RemoteAction("agentic.status", ctx, opts)           // local
r := c.RemoteAction("charon:agentic.dispatch", ctx, opts)   // remote
r := c.RemoteAction("snider.lthn:brain.recall", ctx, opts)  // web3
```

### MCP Auto-Exposure

MCP auto-exposes all registered Actions as tools via `c.Actions()`. Register an Action and it appears as an MCP tool. The API stream primitive (`c.API()`) handles transport.

---

## 4. Package Structure

```
cmd/core-agent/main.go       — entry point: core.New + Run
pkg/agentic/                  — orchestration (dispatch, prep, verify, scan, commands)
pkg/agentic/actions.go        — named Action handlers (ctx, Options) -> Result
pkg/agentic/proc.go           — process helpers via s.Core().Process()
pkg/agentic/handlers.go       — IPC completion pipeline handlers
pkg/agentic/status.go         — workspace status (WriteAtomic + JSONMarshalString)
pkg/agentic/paths.go          — paths, fs (NewUnrestricted), helpers
pkg/agentic/dispatch.go       — agent dispatch logic
pkg/agentic/prep.go           — workspace preparation
pkg/agentic/scan.go           — Forge scanning for work
pkg/agentic/epic.go           — epic creation
pkg/agentic/pr.go             — pull request management
pkg/agentic/plan.go           — plan CRUD
pkg/agentic/queue.go          — dispatch queue
pkg/agentic/runner.go         — runner service (concurrency, drain)
pkg/agentic/verify.go         — output verification
pkg/agentic/watch.go          — workspace watcher
pkg/agentic/resume.go         — session resumption
pkg/agentic/review_queue.go   — review queue management
pkg/agentic/mirror.go         — remote mirroring
pkg/agentic/remote.go         — remote dispatch
pkg/agentic/shutdown.go       — graceful shutdown
pkg/agentic/events.go         — event definitions
pkg/agentic/transport.go      — Forgejo HTTP client (one file)
pkg/agentic/commands.go       — CLI command registration
pkg/brain/                    — OpenBrain (recall, remember, search)
pkg/brain/brain.go            — brain service
pkg/brain/direct.go           — direct API calls
pkg/brain/messaging.go        — agent-to-agent messaging
pkg/brain/provider.go         — embedding provider
pkg/brain/register.go         — service registration
pkg/brain/tools.go            — MCP tool handlers
pkg/lib/                      — embedded templates, personas, flows, plans
pkg/messages/                 — typed message structs for IPC broadcast
pkg/monitor/                  — agent monitoring via IPC (ServiceRuntime)
pkg/setup/                    — workspace detection + scaffolding (Service)
claude/                       — Claude Code plugin definitions
docs/                         — RFC, plans, architecture
```

---

## 5. Process Execution

All commands via `s.Core().Process()`. Returns `core.Result` — Value is always a string.

```go
func (s *PrepSubsystem) runCmd(ctx context.Context, dir, command string, args ...string) core.Result {
    return s.Core().Process().RunIn(ctx, dir, command, args...)
}

func (s *PrepSubsystem) runCmdOK(ctx context.Context, dir, command string, args ...string) bool {
    return s.runCmd(ctx, dir, command, args...).OK
}

func (s *PrepSubsystem) gitCmd(ctx context.Context, dir string, args ...string) core.Result {
    return s.runCmd(ctx, dir, "git", args...)
}

func (s *PrepSubsystem) gitOutput(ctx context.Context, dir string, args ...string) string {
    r := s.gitCmd(ctx, dir, args...)
    if !r.OK { return "" }
    return core.Trim(r.Value.(string))
}
```

go-process is fully Result-native. `Start`, `Run`, `StartWithOptions`, `RunWithOptions` all return `core.Result`. Value is `*Process` for Start, `string` for Run. OK=true guarantees the type.

---

## 6. Status Management

Workspace status uses `WriteAtomic` + `JSONMarshalString` for safe concurrent access:

```go
func writeStatus(wsDir string, status *WorkspaceStatus) error {
    status.UpdatedAt = time.Now()
    statusPath := core.JoinPath(wsDir, "status.json")
    if r := fs.WriteAtomic(statusPath, core.JSONMarshalString(status)); !r.OK {
        err, _ := r.Value.(error)
        return core.E("writeStatus", "failed to write status", err)
    }
    return nil
}
```

### Registry for Workspace Tracking

```go
workspaces := core.NewRegistry[*WorkspaceStatus]()
workspaces.Set(wsDir, status)
workspaces.Get(wsDir)
workspaces.Each(func(dir string, st *WorkspaceStatus) { ... })
workspaces.Names()  // insertion order
c.RegistryOf("actions").List("agentic.*")
```

### Filesystem

Package-level unrestricted Fs via Core primitive:

```go
var fs = (&core.Fs{}).NewUnrestricted()
```

---

## 7. Monitor Service

Embeds `*core.ServiceRuntime[MonitorOptions]`. All notifications via `m.Core().ACTION(messages.X{})` — no ChannelNotifier interface. Git operations via `m.Core().Process()`.

```go
func Register(c *core.Core) core.Result {
    mon := New()
    mon.ServiceRuntime = core.NewServiceRuntime(c, MonitorOptions{})

    c.RegisterAction(func(c *core.Core, msg core.Message) core.Result {
        switch ev := msg.(type) {
        case messages.AgentCompleted:
            mon.handleAgentCompleted(ev)
        case messages.AgentStarted:
            mon.handleAgentStarted(ev)
        }
        return core.Result{OK: true}
    })

    return core.Result{Value: mon, OK: true}
}
```

### IPC Completion Pipeline

Registered in `RegisterHandlers()`:

```
AgentCompleted -> QA handler -> QAResult
QAResult{Passed} -> PR handler -> PRCreated
PRCreated -> Verify handler -> PRMerged | PRNeedsReview
AgentCompleted -> Ingest handler (findings -> issues)
AgentCompleted -> Poke handler (drain queue)
```

All handlers use `c.ACTION(messages.X{})` — no ChannelNotifier, no callbacks.

---

## 8. MCP Tools

25+ tools registered via named Actions:

### Dispatch
`agentic_dispatch`, `agentic_status`, `agentic_scan`, `agentic_watch`, `agentic_resume`, `agentic_review_queue`, `agentic_dispatch_start`, `agentic_dispatch_shutdown`

### Workspace
`agentic_prep_workspace`, `agentic_create_epic`, `agentic_create_pr`, `agentic_list_prs`, `agentic_mirror`

### Plans
`agentic_plan_create`, `agentic_plan_read`, `agentic_plan_update`, `agentic_plan_list`, `agentic_plan_delete`

### Brain
`brain_remember`, `brain_recall`, `brain_forget`

### Messaging
`agent_send`, `agent_inbox`, `agent_conversation`

---

## 9. Runner Service

Owns dispatch concurrency (from `agents.yaml` config) and queue drain.

- Checks concurrency limits (total + per-model) before dispatching
- Checks rate limits (daily, min_delay, burst window)
- Pops next queued task matching an available pool
- Spawns agent in sandboxed workspace
- Channel notifications: `AgentStarted`/`AgentCompleted` push to Claude Code sessions

---

## 10. Dispatch and Pool Routing

### agents.yaml

See `code/core/agent/RFC.md` section "Configuration" for the full agents.yaml schema.

Go loads this config during `Register()`:

```go
cfg := prep.loadAgentsConfig()
c.Config().Set("agents.concurrency", cfg.Concurrency)
c.Config().Set("agents.rates", cfg.Rates)
```

### Configuration Access

```go
c.Config().Set("agents.concurrency", 5)
c.Config().String("workspace.root")
c.Config().Int("agents.concurrency")
c.Config().Enable("auto-merge")
if c.Config().Enabled("auto-merge") { ... }
```

### Workspace Prep by Language

- **Go**: `go mod download`, `go work sync`
- **PHP**: `composer install`
- **TypeScript**: `npm install`
- Language-specific CODEX.md generation from RFC

---

## 11. Quality Gates

### Banned Imports

Source files (not tests) must not import these — Core provides alternatives:

| Banned | Replacement |
|--------|-------------|
| `"os"` | `core.Env`, `core.Fs` |
| `"os/exec"` | `s.Core().Process()` |
| `"io"` | `core.ReadAll`, `core.WriteAll` |
| `"fmt"` | `core.Println`, `core.Sprintf`, `core.Concat` |
| `"errors"` | `core.E()` |
| `"log"` | `core.Info`, `core.Error`, `core.Security` |
| `"encoding/json"` | `core.JSONMarshalString`, `core.JSONUnmarshalString` |
| `"path/filepath"` | `core.JoinPath`, `core.Path` |
| `"unsafe"` | (never) |
| `"strings"` | `core.Contains`, `core.Split`, `core.Trim` |

Verification:

```bash
grep -rn '"os"\|"os/exec"\|"io"\|"fmt"\|"errors"\|"log"\|"encoding/json"\|"path/filepath"\|"unsafe"\|"strings"' *.go **/*.go \
  | grep -v _test.go
```

### Error Handling

All errors via `core.E()`. All logging via Core:

```go
return core.E("dispatch.prep", "workspace not found", nil)
return core.E("dispatch.prep", core.Concat("repo ", repo, " invalid"), cause)
core.Info("agent dispatched", "repo", repo, "agent", agent)
core.Error("dispatch failed", "err", err)
core.Security("entitlement.denied", "action", action, "reason", reason)
```

### String Operations

No `fmt`, no `strings`, no `+` concat:

```go
core.Println(value)                    // not fmt.Println
core.Sprintf("port: %d", port)        // not fmt.Sprintf
core.Concat("hello ", name)            // not "hello " + name
core.Path(dir, "status.json")         // not dir + "/status.json"
core.Contains(s, "prefix")            // not strings.Contains
core.Split(s, "/")                    // not strings.Split
core.Trim(s)                          // not strings.TrimSpace
```

### JSON Serialisation

All JSON via Core primitives:

```go
data := core.JSONMarshalString(status)
core.JSONUnmarshalString(jsonStr, &result)
```

### Validation and IDs

```go
if r := core.ValidateName(input.Repo); !r.OK { return r }
safe := core.SanitisePath(userInput)
id := core.ID()  // "id-42-a3f2b1"
```

### Stream Helpers and Data

```go
r := c.Data().ReadString("prompts/coding.md")
c.Data().List("templates/")
c.Drive().New(core.NewOptions(
    core.Option{Key: "name", Value: "charon"},
    core.Option{Key: "transport", Value: "http://10.69.69.165:9101"},
))
```

### Comments (AX Principle 2)

Every exported function MUST have a usage-example comment:

```go
// gitCmd runs a git command in a directory.
//
//   r := s.gitCmd(ctx, "/repo", "log", "--oneline")
func (s *PrepSubsystem) gitCmd(ctx context.Context, dir string, args ...string) core.Result {
```

### Test Strategy (AX Principle 7)

`TestFile_Function_{Good,Bad,Ugly}` — 100% naming compliance target.

Verification:

```bash
grep -rn "^func Test" *_test.go **/*_test.go \
  | grep -v "Test[A-Z][a-z]*_.*_\(Good\|Bad\|Ugly\)"
```

---

## 12. Reference Material

| Resource | Location |
|----------|----------|
| Agent contract (cross-cutting) | `code/core/agent/RFC.md` |
| Core framework spec | `code/core/go/RFC.md` |
| Process primitives | `code/core/go/process/RFC.md` |
| MCP spec | `code/core/mcp/RFC.md` |
| PHP implementation | `code/core/php/agent/RFC.md` |

---

## Changelog

- 2026-03-29: Restructured as Go implementation spec. Language-agnostic contract moved to `code/core/agent/RFC.md`. Retained all Go-specific patterns (ServiceRuntime, core.E, banned imports, AX principles).
- 2026-03-27: Initial Go agent RFC with MCP tools, runner service, fleet mode, polyglot mapping.
