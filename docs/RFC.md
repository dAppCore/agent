# core/agent API Contract — RFC Specification

> `dappco.re/go/core/agent` — Agentic dispatch, orchestration, and pipeline management.
> An agent should be able to understand core/agent's architecture from this document alone.

**Status:** v0.8.0+alpha.1
**Module:** `dappco.re/go/core/agent`
**Depends on:** core/go v0.8.0, go-process v0.8.0

---

## 1. Purpose

core/agent dispatches AI agents (Claude, Codex, Gemini) to work on tasks in sandboxed git worktrees, monitors their progress, verifies output, and manages the merge pipeline.

core/go provides the primitives. core/agent composes them.

### File Layout

```
cmd/core-agent/main.go       — entry point: core.New + Run
pkg/agentic/                  — orchestration (dispatch, prep, verify, scan, commands)
pkg/agentic/actions.go        — named Action handlers (ctx, Options) → Result
pkg/agentic/pid.go            — PID lifecycle helpers
pkg/agentic/handlers.go       — IPC completion pipeline handlers
pkg/agentic/status.go         — workspace status (WriteAtomic + JSONMarshalString)
pkg/agentic/paths.go          — paths, fs (NewUnrestricted), helpers
pkg/brain/                    — OpenBrain (recall, remember, search)
pkg/lib/                      — embedded templates, personas, flows, plans
pkg/messages/                 — typed message structs for IPC broadcast
pkg/monitor/                  — agent monitoring via IPC (ServiceRuntime)
pkg/setup/                    — workspace detection + scaffolding (Service)
claude/                       — Claude Code plugin definitions
docs/                         — RFC, plans, architecture
```

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

---

## 3. Named Actions — The Capability Map

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
        Description: "QA → PR → Verify → Merge",
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

---

## 4. Completion Pipeline

When an agent completes, the IPC handler chain fires. Registered in `RegisterHandlers()`:

```
AgentCompleted → QA handler → QAResult
QAResult{Passed} → PR handler → PRCreated
PRCreated → Verify handler → PRMerged | PRNeedsReview
AgentCompleted → Ingest handler (findings → issues)
AgentCompleted → Poke handler (drain queue)
```

All handlers use `c.ACTION(messages.X{})` — no ChannelNotifier, no callbacks.

---

## 5. Process Execution

All commands via `s.Core().Process()`. Returns `core.Result` — Value is always a string.

```go
process := s.Core().Process()
r := process.RunIn(ctx, dir, "git", "log", "--oneline", "-20")
if r.OK {
    output := core.Trim(r.Value.(string))
}

r = process.RunWithEnv(ctx, dir, []string{"GOWORK=off"}, "go", "test", "./...")
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

---

## 7. Filesystem

No `unsafe.Pointer`. Package-level unrestricted Fs via Core primitive:

```go
var fs = (&core.Fs{}).NewUnrestricted()
```

---

## 8. IPC Messages

All inter-service communication via typed messages in `pkg/messages/`:

```go
// Agent lifecycle
messages.AgentStarted{Agent, Repo, Workspace}
messages.AgentCompleted{Agent, Repo, Workspace, Status}

// Pipeline
messages.QAResult{Workspace, Repo, Passed}
messages.PRCreated{Repo, Branch, PRURL, PRNum}
messages.PRMerged{Repo, PRURL, PRNum}
messages.PRNeedsReview{Repo, PRURL, PRNum, Reason}

// Queue
messages.QueueDrained{Completed}
messages.PokeQueue{}

// Monitor
messages.HarvestComplete{Repo, Branch, Files}
messages.HarvestRejected{Repo, Branch, Reason}
messages.InboxMessage{New, Total}
```

---

## 9. Monitor

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

---

## 10. Setup

Service with `*core.ServiceRuntime[SetupOptions]`. Detects project type, generates configs, scaffolds workspaces.

```go
func Register(c *core.Core) core.Result {
    svc := &Service{
        ServiceRuntime: core.NewServiceRuntime(c, SetupOptions{}),
    }
    return core.Result{Value: svc, OK: true}
}
```

---

## 11. Entitlements

Actions are gated by `c.Entitled()` — checked automatically in `Action.Run()`.

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

---

## 12. MCP — Action Aggregator

MCP auto-exposes all registered Actions as tools via `c.Actions()`. Register an Action → it appears as an MCP tool. The API stream primitive (`c.API()`) handles transport.

---

## 13. Remote Dispatch

Transparent local/remote via `host:action` syntax:

```go
r := c.RemoteAction("agentic.status", ctx, opts)           // local
r := c.RemoteAction("charon:agentic.dispatch", ctx, opts)   // remote
r := c.RemoteAction("snider.lthn:brain.recall", ctx, opts)  // web3
```

---

## 14. Quality Gates

```bash
# No disallowed imports (source files only)
grep -rn '"os"\|"os/exec"\|"io"\|"fmt"\|"errors"\|"log"\|"encoding/json"\|"path/filepath"\|"unsafe"\|"strings"' *.go **/*.go \
  | grep -v _test.go

# Test naming: TestFile_Function_{Good,Bad,Ugly}
grep -rn "^func Test" *_test.go **/*_test.go \
  | grep -v "Test[A-Z][a-z]*_.*_\(Good\|Bad\|Ugly\)"
```

---

## 15. Validation and IDs

```go
if r := core.ValidateName(input.Repo); !r.OK { return r }
safe := core.SanitisePath(userInput)
id := core.ID()  // "id-42-a3f2b1"
```

---

## 16. JSON Serialisation

All JSON via Core primitives. No `encoding/json` import.

```go
data := core.JSONMarshalString(status)
core.JSONUnmarshalString(jsonStr, &result)
```

---

## 17. Configuration

```go
c.Config().Set("agents.concurrency", 5)
c.Config().String("workspace.root")
c.Config().Int("agents.concurrency")
c.Config().Enable("auto-merge")
if c.Config().Enabled("auto-merge") { ... }
```

---

## 18. Registry

Use `Registry[T]` for any named collection. No `map[string]*T + sync.Mutex`.

```go
workspaces := core.NewRegistry[*WorkspaceStatus]()
workspaces.Set(wsDir, status)
workspaces.Get(wsDir)
workspaces.Each(func(dir string, st *WorkspaceStatus) { ... })
workspaces.Names()  // insertion order
c.RegistryOf("actions").List("agentic.*")
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

## 20. Error Handling and Logging

All errors via `core.E()`. All logging via Core. No `fmt`, `errors`, or `log` imports.

```go
return core.E("dispatch.prep", "workspace not found", nil)
return core.E("dispatch.prep", core.Concat("repo ", repo, " invalid"), cause)
core.Info("agent dispatched", "repo", repo, "agent", agent)
core.Error("dispatch failed", "err", err)
core.Security("entitlement.denied", "action", action, "reason", reason)
```

---

## 21. Stream Helpers and Data

```go
r := c.Data().ReadString("prompts/coding.md")
c.Data().List("templates/")
c.Drive().New(core.NewOptions(
    core.Option{Key: "name", Value: "charon"},
    core.Option{Key: "transport", Value: "http://10.69.69.165:9101"},
))
```

---

## 22. Comments (AX Principle 2)

Every exported function MUST have a usage-example comment:

```go
// Process runs a git command in a directory.
//
//   r := s.Core().Process().RunIn(ctx, "/repo", "git", "log", "--oneline")
```

---

## 23. Test Strategy (AX Principle 7)

`TestFile_Function_{Good,Bad,Ugly}` — 100% naming compliance target.

---

## Consumer RFCs

| Package | RFC | Role |
|---------|-----|------|
| core/go | `core/go/docs/RFC.md` | Primitives — all 21 sections |
| go-process | `core/go-process/docs/RFC.md` | Process Action handlers (Result-native) |

---

## Changelog

- 2026-03-30: `pkg/agentic/commands_workspace.go` now has a matching example companion, closing the last agentic source file without example coverage.
- 2026-03-30: plan files and review queue rate-limit state now use `WriteAtomic`, keeping JSON state writes aligned with the AX safe-write convention.
- 2026-03-30: plan create tests now assert the documented `core.ID()` shape and repeated plan creation produces unique IDs, keeping the plan contract aligned with the simplified generator.
- 2026-03-30: dispatch completion monitoring now uses a named helper instead of an inline Action closure, keeping the spawned-process finaliser AX-native.
- 2026-03-30: lib task bundle and recursive embed traversal now use `JoinPath` for filesystem paths, removing the last string-concatenated path joins in `pkg/lib`.
- 2026-03-30: runner workspace status projections now use explicit typed copies, and `ReadStatusResult` gained direct AX-7 coverage in both runner and agentic packages.
- 2026-03-30: transport helpers preserve request and read causes, brain direct API calls surface upstream bodies, and review queue retry parsing no longer uses `MustCompile`.
- 2026-03-30: direct Core process calls replaced the `proc.go` wrapper layer; PID helpers now live in `pid.go` and the workspace template documents `c.Process()` directly.
- 2026-03-30: main now logs startup failures with structured context, and the workspace contract reference restored usage-example comments for the Action lifecycle messages.
- 2026-03-30: plan IDs now come from core.ID(), workspace prep validates org/repo names with core.ValidateName, and plan paths use core.SanitisePath.
- 2026-03-29: cmd/core-agent no longer rewrites `os.Args` before startup. The binary-owned commands now use named handlers, keeping the entrypoint on Core CLI primitives instead of repo-local argument mutation.
- 2026-03-29: brain/provider.go no longer imports net/http for Gin handlers. Handler responses now use named status constants and shared response helpers. HTTP remains intentionally centralised in pkg/agentic/transport.go.
- 2026-03-26: WIP — net/http consolidated to transport.go (ONE file). net/url + io/fs eliminated. RFC-025 updated with 3 new quality gates (net/http, net/url, io/fs). 1:1 test + example test coverage. Array[T].Deduplicate replaces custom helpers.
- 2026-03-25: Quality gates pass. Zero disallowed imports (all 10). encoding/json→Core JSON. path/filepath→Core Path. os→Core Env/Fs. io→Core ReadAll/WriteAll. go-process fully Result-native. ServiceRuntime on all subsystems. 22 named Actions + Task pipeline. ChannelNotifier→IPC. Reference docs synced.
- 2026-03-25: Initial spec — written with full core/go v0.8.0 domain context.
