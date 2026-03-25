# core/agent API Contract — RFC Specification

> `dappco.re/go/core/agent` — Agentic dispatch, orchestration, and pipeline management.
> An agent should be able to understand core/agent's architecture from this document alone.

**Status:** v0.8.0
**Module:** `dappco.re/go/core/agent`
**Depends on:** core/go v0.8.0, go-process v0.8.0

---

## 1. Purpose

core/agent dispatches AI agents (Claude, Codex, Gemini) to work on tasks in sandboxed git worktrees, monitors their progress, verifies output, and manages the merge pipeline.

core/go provides the primitives. core/agent composes them.

### Current State (2026-03-25)

The codebase is PRE-migration. The RFC describes the v0.8.0 target. What exists today:

- `pkg/agentic/proc.go` — standalone process helpers with `ensureProcess()`. **Delete** — replace with `s.Core().Process()`
- `pkg/agentic/handlers.go` — nested `c.ACTION()` cascade 4 levels deep. **Replace** with `c.Task("agent.completion")`
- `pkg/agentic/commands.go` — closures already extracted to named methods (done in prior session)
- `pkg/agentic/commands_forge.go` — forge command methods (done)
- `pkg/agentic/commands_workspace.go` — workspace command methods (done)
- `pkg/agentic/dispatch.go` — `spawnAgent` decomposed into 7 functions (done)
- `pkg/agentic/status.go` — uses `os.WriteFile` for status.json. **Replace** with `Fs.WriteAtomic`
- `pkg/agentic/paths.go` — uses `unsafe.Pointer` to bypass Fs.root. **Replace** with `Fs.NewUnrestricted()`
- `pkg/messages/` — typed IPC message structs (`AgentCompleted`, `QAResult`, etc.)
- `pkg/brain/` — OpenBrain integration (recall/remember)
- `pkg/monitor/` — agent monitoring + notifications
- `pkg/setup/` — workspace scaffolding
- `OnStartup`/`OnShutdown` — currently return `error`. **Change** to return `Result`

### File Layout

```
cmd/core-agent/main.go       — entry point: core.New + Run
pkg/agentic/                  — orchestration (dispatch, prep, verify, scan, commands)
pkg/agentic/proc.go           — DELETE (replace with c.Process())
pkg/agentic/handlers.go       — REWRITE (cascade → Task pipeline)
pkg/agentic/status.go         — MIGRATE (os.WriteFile → WriteAtomic)
pkg/agentic/paths.go          — MIGRATE (unsafe.Pointer → NewUnrestricted)
pkg/brain/                    — OpenBrain (recall, remember, search)
pkg/lib/                      — embedded templates, personas, flows, plans
pkg/messages/                 — typed message structs for IPC broadcast
pkg/monitor/                  — agent monitoring + channel notifications
pkg/setup/                    — workspace detection + scaffolding
claude/                       — Claude Code plugin definitions
docs/                         — RFC, plans, architecture
```

---

## 2. Service Registration

```go
func Register(c *core.Core) core.Result {
    svc := &PrepSubsystem{
        ServiceRuntime: core.NewServiceRuntime(c, AgentOptions{}),
    }
    return core.Result{Value: svc, OK: true}
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

---

## 3. Named Actions — The Capability Map

All capabilities registered as named Actions during OnStartup. Inspectable, composable, gatable by Entitlements.

```go
func (s *PrepSubsystem) OnStartup(ctx context.Context) core.Result {
    c := s.Core()

    // Dispatch
    c.Action("agentic.dispatch", s.handleDispatch)
    c.Action("agentic.prep", s.handlePrep)
    c.Action("agentic.status", s.handleStatus)
    c.Action("agentic.resume", s.handleResume)
    c.Action("agentic.scan", s.handleScan)

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

    // Brain
    c.Action("brain.recall", s.handleBrainRecall)
    c.Action("brain.remember", s.handleBrainRemember)

    // Completion pipeline
    c.Task("agent.completion", core.Task{
        Description: "QA → PR → Verify → Merge",
        Steps: []core.Step{
            {Action: "agentic.qa"},
            {Action: "agentic.auto-pr"},
            {Action: "agentic.verify"},
            {Action: "agentic.ingest", Async: true},
            {Action: "agentic.poke", Async: true},
        },
    })

    s.registerCommands(ctx)
    return core.Result{OK: true}
}
```

---

## 4. Completion Pipeline

When an agent completes, the Task runs sequentially. Async steps fire without blocking the queue drain.

```go
c.RegisterAction(func(c *core.Core, msg core.Message) core.Result {
    if ev, ok := msg.(messages.AgentCompleted); ok {
        opts := core.NewOptions(
            core.Option{Key: "repo", Value: ev.Repo},
            core.Option{Key: "workspace", Value: ev.Workspace},
        )
        c.PerformAsync("agent.completion", opts)
    }
    return core.Result{OK: true}
})
```

Steps: QA (build+test) → Auto-PR (git push + Forge API) → Verify (test + merge).
Ingest and Poke run async — Poke drains the queue immediately.

---

## 5. Process Execution

All commands via `c.Process()`. No `os/exec`, no `proc.go`, no `ensureProcess()`.

```go
// Git operations
func (s *PrepSubsystem) gitCmd(ctx context.Context, dir string, args ...string) core.Result {
    return s.Core().Process().RunIn(ctx, dir, "git", args...)
}

func (s *PrepSubsystem) gitOK(ctx context.Context, dir string, args ...string) bool {
    return s.gitCmd(ctx, dir, args...).OK
}

func (s *PrepSubsystem) gitOutput(ctx context.Context, dir string, args ...string) string {
    r := s.gitCmd(ctx, dir, args...)
    if !r.OK { return "" }
    return core.Trim(r.Value.(string))
}
```

---

## 6. Status Management

Workspace status uses `WriteAtomic` for safe concurrent access + per-workspace mutex for read-modify-write:

```go
// Write
s.Core().Fs().WriteAtomic(statusPath, core.JSONMarshalString(status))

// Read-modify-write with lock
s.withLock(wsDir, func() {
    var st WorkspaceStatus
    core.JSONUnmarshalString(s.Core().Fs().Read(statusPath).Value.(string), &st)
    st.Status = "completed"
    s.Core().Fs().WriteAtomic(statusPath, core.JSONMarshalString(st))
})
```

---

## 7. Filesystem

No `unsafe.Pointer`. Sandboxed by default, unrestricted when needed:

```go
// Sandboxed to workspace
f := (&core.Fs{}).New(workspaceDir)

// Full access when required
f := s.Core().Fs().NewUnrestricted()
```

---

## 8. Validation and IDs

```go
// Validate input
if r := core.ValidateName(input.Repo); !r.OK { return r }
safe := core.SanitisePath(userInput)

// Generate unique identifiers
id := core.ID()  // "id-42-a3f2b1"
```

---

## 9. Entitlements

Actions are gated by `c.Entitled()` — checked automatically in `Action.Run()`.

For explicit gating with quantity checks:

```go
func (s *PrepSubsystem) handleDispatch(ctx context.Context, opts core.Options) core.Result {
    // Concurrency limit
    e := s.Core().Entitled("agentic.concurrency", 1)
    if !e.Allowed {
        return core.Result{Value: core.E("dispatch", e.Reason, nil), OK: false}
    }

    // ... dispatch agent ...

    s.Core().RecordUsage("agentic.dispatch")
    return core.Result{OK: true}
}
```

Enables: SaaS tier gating, usage tracking, workspace isolation.

---

## 10. MCP — Action Aggregator

MCP auto-exposes all registered Actions as tools:

```go
func (s *MCPService) OnStartup(ctx context.Context) core.Result {
    for _, name := range s.Core().Actions() {
        name := name  // capture loop variable
        action := s.Core().Action(name)
        s.server.AddTool(mcp.Tool{
            Name:        name,
            Description: action.Description,
            InputSchema: schemaFromOptions(action.Schema),
            Handler: func(ctx context.Context, input map[string]any) (any, error) {
                // Re-resolve action at call time (not captured pointer)
                r := s.Core().Action(name).Run(ctx, optionsFromInput(input))
                if !r.OK { return nil, r.Value.(error) }
                return r.Value, nil
            },
        })
    }
    return core.Result{OK: true}
}
```

Register an Action → it appears as an MCP tool. No hand-wiring.

---

## 11. Remote Dispatch

Transparent local/remote via `host:action` syntax:

```go
// Local
r := c.RemoteAction("agentic.status", ctx, opts)

// Remote — same API
r := c.RemoteAction("charon:agentic.dispatch", ctx, opts)

// Web3
r := c.RemoteAction("snider.lthn:brain.recall", ctx, opts)
```

---

## 12. JSON Serialisation

All JSON via Core primitives. No `encoding/json` import.

```go
data := core.JSONMarshalString(status)
core.JSONUnmarshal(responseBytes, &result)
```

---

## 13. Test Strategy

AX-7: `TestFile_Function_{Good,Bad,Ugly}` — 100% naming compliance target.

```
TestHandlers_CompletionPipeline_Good     — QA+PR+Verify succeed, Poke fires
TestHandlers_CompletionPipeline_Bad      — QA fails, chain stops
TestHandlers_CompletionPipeline_Ugly     — handler panics, pipeline recovers
TestDispatch_Entitlement_Good            — entitled workspace dispatches
TestDispatch_Entitlement_Bad             — denied workspace gets error
TestPrep_GitCmd_Good                     — via c.Process()
TestStatus_WriteAtomic_Ugly              — concurrent writes don't corrupt
TestMCP_ActionAggregator_Good            — Actions appear as MCP tools
```

---

## 14. Error Handling and Logging

All errors via `core.E()`. All logging via Core. No `fmt`, `errors`, or `log` imports.

```go
// Structured errors
return core.E("dispatch.prep", "workspace not found", nil)
return core.E("dispatch.prep", core.Concat("repo ", repo, " invalid"), cause)

// Error inspection
core.Operation(err)      // "dispatch.prep"
core.ErrorMessage(err)   // "workspace not found"
core.Root(err)           // unwrap to root cause

// Logging
core.Info("agent dispatched", "repo", repo, "agent", agent)
core.Warn("queue full", "pending", count)
core.Error("dispatch failed", "err", err)
core.Security("entitlement.denied", "action", action, "reason", reason)
```

---

## 15. Configuration

```go
// Runtime settings
c.Config().Set("agents.concurrency", 5)
c.Config().String("workspace.root")
c.Config().Int("agents.concurrency")

// Feature flags
c.Config().Enable("auto-merge")
if c.Config().Enabled("auto-merge") { ... }
```

---

## 16. Registry

Use `Registry[T]` for any named collection. No `map[string]*T + sync.Mutex`.

```go
// Workspace status tracking
workspaces := core.NewRegistry[*WorkspaceStatus]()
workspaces.Set(wsDir, status)
workspaces.Get(wsDir)
workspaces.Each(func(dir string, st *WorkspaceStatus) { ... })
workspaces.Names()  // insertion order

// Cross-cutting queries via Core
c.RegistryOf("actions").List("agentic.*")
c.RegistryOf("services").Names()
```

---

## 17. Stream Helpers

No `io` import. Core wraps all stream operations:

```go
// Read entire stream
r := c.Fs().ReadStream(path)
content := core.ReadAll(r.Value)

// Write to stream
w := c.Fs().WriteStream(path)
core.WriteAll(w.Value, data)

// Close any stream
core.CloseStream(handle)
```

---

## 18. Data and Drive

```go
// Embedded assets (prompts, templates, personas)
r := c.Data().ReadString("prompts/coding.md")
c.Data().List("templates/")
c.Data().Mounts()  // all mounted asset namespaces

// Transport configuration
c.Drive().New(core.NewOptions(
    core.Option{Key: "name", Value: "charon"},
    core.Option{Key: "transport", Value: "http://10.69.69.165:9101"},
))
c.Drive().Get("charon")
```

---

## 19. String Operations

No `fmt`, no `strings`, no `+` concat. Core provides everything:

```go
core.Println(value)                    // not fmt.Println
core.Sprintf("port: %d", port)        // not fmt.Sprintf
core.Concat("hello ", name)            // not "hello " + name
core.Path(dir, "status.json")         // not dir + "/status.json"
core.Contains(s, "prefix")            // not strings.Contains
core.Split(s, "/")                    // not strings.Split
core.Trim(s)                          // not strings.TrimSpace
```

---

## 20. Comments (AX Principle 2)

Every exported function MUST have a usage-example comment:

```go
// gitCmd runs a git command in a directory.
//
//   r := s.gitCmd(ctx, "/repo", "log", "--oneline")
func (s *PrepSubsystem) gitCmd(ctx context.Context, dir string, args ...string) core.Result {
```

No exceptions. The comment is for every model that will ever read the code.

---

## 21. Example Tests (AX Principle 7b)

One `{source}_example_test.go` per source file. Examples serve as test + documentation + godoc.

```go
// file: dispatch_example_test.go

func ExamplePrepSubsystem_handleDispatch() {
    c := core.New(core.WithService(agentic.Register))
    r := c.Action("agentic.dispatch").Run(ctx, opts)
    core.Println(r.OK)
    // Output: true
}
```

---

## 22. Quality Gates (AX Principle 9)

```bash
# No disallowed imports (all 10)
grep -rn '"os"\|"os/exec"\|"io"\|"fmt"\|"errors"\|"log"\|"encoding/json"\|"path/filepath"\|"unsafe"\|"strings"' *.go **/*.go \
  | grep -v _test.go

# Test naming
grep -rn "^func Test" *_test.go **/*_test.go \
  | grep -v "Test[A-Z][a-z]*_.*_\(Good\|Bad\|Ugly\)"

# String concat
grep -rn '" + \| + "' *.go **/*.go | grep -v _test.go | grep -v "//"
```

---

## Consumer RFCs

| Package | RFC | Role |
|---------|-----|------|
| core/go | `core/go/docs/RFC.md` | Primitives — all 21 sections |
| go-process | `core/go-process/docs/RFC.md` | Process Action handlers |

---

## Changelog

- 2026-03-25: Initial spec — written with full core/go v0.8.0 domain context.
