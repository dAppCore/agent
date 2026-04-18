# RFC-025: Agent Experience (AX) Design Principles

- **Status:** Active
- **Authors:** Snider, Cladius
- **Date:** 2026-03-25
- **Applies to:** All Core ecosystem packages (CoreGO, CorePHP, CoreTS, core-agent)

## Abstract

Agent Experience (AX) is a design paradigm for software systems where the primary code consumer is an AI agent, not a human developer. AX sits alongside User Experience (UX) and Developer Experience (DX) as the third era of interface design.

This RFC establishes AX as a formal design principle for the Core ecosystem and defines the conventions that follow from it.

## Motivation

As of early 2026, AI agents write, review, and maintain the majority of code in the Core ecosystem. The original author has not manually edited code (outside of Core struct design) since October 2025. Code is processed semantically — agents reason about intent, not characters.

Design patterns inherited from the human-developer era optimise for the wrong consumer:

- **Short names** save keystrokes but increase semantic ambiguity
- **Functional option chains** are fluent for humans but opaque for agents tracing configuration
- **Error-at-every-call-site** produces 50% boilerplate that obscures intent
- **Generic type parameters** force agents to carry type context that the runtime already has
- **Panic-hiding conventions** (`Must*`) create implicit control flow that agents must special-case
- **Raw exec.Command** bypasses Core primitives — untestable, no entitlement check, path traversal risk

AX acknowledges this shift and provides principles for designing code, APIs, file structures, and conventions that serve AI agents as first-class consumers.

## The Three Eras

| Era | Primary Consumer | Optimises For | Key Metric |
|-----|-----------------|---------------|------------|
| UX | End users | Discoverability, forgiveness, visual clarity | Task completion time |
| DX | Developers | Typing speed, IDE support, convention familiarity | Time to first commit |
| AX | AI agents | Predictability, composability, semantic navigation | Correct-on-first-pass rate |

AX does not replace UX or DX. End users still need good UX. Developers still need good DX. But when the primary code author and maintainer is an AI agent, the codebase should be designed for that consumer first.

## Principles

### 1. Predictable Names Over Short Names

Names are tokens that agents pattern-match across languages and contexts. Abbreviations introduce mapping overhead.

```
Config    not  Cfg
Service   not  Srv
Embed     not  Emb
Error     not  Err (as a subsystem name; err for local variables is fine)
Options   not  Opts
```

**Rule:** If a name would require a comment to explain, it is too short.

**Exception:** Industry-standard abbreviations that are universally understood (`HTTP`, `URL`, `ID`, `IPC`, `I18n`) are acceptable. The test: would an agent trained on any mainstream language recognise it without context?

### 2. Comments as Usage Examples

The function signature tells WHAT. The comment shows HOW with real values.

```go
// Entitled checks if an action is permitted.
//
//   e := c.Entitled("process.run")
//   e := c.Entitled("social.accounts", 3)
//   if e.Allowed { proceed() }

// WriteAtomic writes via temp file then rename (safe for concurrent readers).
//
//   r := fs.WriteAtomic("/status.json", data)

// Action registers or invokes a named callable.
//
//   c.Action("git.log", handler)           // register
//   c.Action("git.log").Run(ctx, opts)     // invoke
```

**Rule:** If a comment restates what the type signature already says, delete it. If a comment shows a concrete usage with realistic values, keep it.

**Rationale:** Agents learn from examples more effectively than from descriptions. A comment like "Run executes the setup process" adds zero information. A comment like `setup.Run(setup.Options{Path: ".", Template: "auto"})` teaches an agent exactly how to call the function.

### 3. Path Is Documentation

File and directory paths should be self-describing. An agent navigating the filesystem should understand what it is looking at without reading a README.

```
pkg/agentic/dispatch.go         — agent dispatch logic
pkg/agentic/handlers.go         — IPC event handlers
pkg/lib/task/bug-fix.yaml       — bug fix plan template
pkg/lib/persona/engineering/     — engineering personas
flow/deploy/to/homelab.yaml     — deploy TO the homelab
template/dir/workspace/default/  — default workspace scaffold
docs/RFC.md                      — authoritative API contract
```

**Rule:** If an agent needs to read a file to understand what a directory contains, the directory naming has failed.

**Corollary:** The unified path convention (folder structure = HTTP route = CLI command = test path) is AX-native. One path, every surface.

### 4. Templates Over Freeform

When an agent generates code from a template, the output is constrained to known-good shapes. When an agent writes freeform, the output varies.

```go
// Template-driven — consistent output
lib.ExtractWorkspace("default", targetDir, &lib.WorkspaceData{
    Repo: "go-io", Branch: "dev", Task: "fix tests", Agent: "codex",
})

// Freeform — variance in output
"write a workspace setup script that..."
```

**Rule:** For any code pattern that recurs, provide a template. Templates are guardrails for agents.

**Scope:** Templates apply to file generation, workspace scaffolding, config generation, and commit messages. They do NOT apply to novel logic — agents should write business logic freeform with the domain knowledge available.

### 5. Declarative Over Imperative

Agents reason better about declarations of intent than sequences of operations.

```yaml
# Declarative — agent sees what should happen
steps:
  - name: build
    flow: tools/docker-build
    with:
      context: "{{ .app_dir }}"
      image_name: "{{ .image_name }}"

  - name: deploy
    flow: deploy/with/docker
    with:
      host: "{{ .host }}"
```

```go
// Imperative — agent must trace execution
cmd := exec.Command("docker", "build", "--platform", "linux/amd64", "-t", imageName, ".")
cmd.Dir = appDir
if err := cmd.Run(); err != nil {
    return core.E("build", "docker build failed", err)
}
```

**Rule:** Orchestration, configuration, and pipeline logic should be declarative (YAML/JSON). Implementation logic should be imperative (Go/PHP/TS). The boundary is: if an agent needs to compose or modify the logic, make it declarative.

Core's `Task` is the Go-native declarative equivalent — a sequence of named Action steps:

```go
c.Task("deploy", core.Task{
    Steps: []core.Step{
        {Action: "docker.build"},
        {Action: "docker.push"},
        {Action: "deploy.ansible", Async: true},
    },
})
```

### 6. Core Primitives — Universal Types and DI

Every component in the ecosystem registers with Core and communicates through Core's primitives. An agent processing any level of the tree sees identical shapes.

#### Creating Core

```go
c := core.New(
    core.WithOption("name", "core-agent"),
    core.WithService(process.Register),
    core.WithService(agentic.Register),
    core.WithService(monitor.Register),
    core.WithService(brain.Register),
    core.WithService(mcp.Register),
)
c.Run()  // or: if err := c.RunE(); err != nil { ... }
```

`core.New()` returns `*Core`. `WithService` registers a factory `func(*Core) Result`. Services auto-discover: name from package path, lifecycle from `Startable`/`Stoppable` (return `Result`). `HandleIPCEvents` is the one remaining magic method — auto-registered via reflection if the service implements it.

#### Service Registration Pattern

```go
// Service factory — receives Core, returns Result
func Register(c *core.Core) core.Result {
    svc := &MyService{
        ServiceRuntime: core.NewServiceRuntime(c, MyOptions{}),
    }
    return core.Result{Value: svc, OK: true}
}
```

#### Core Subsystem Accessors

| Accessor | Purpose |
|----------|---------|
| `c.Options()` | Input configuration |
| `c.App()` | Application metadata (name, version) |
| `c.Config()` | Runtime settings, feature flags |
| `c.Data()` | Embedded assets (Registry[*Embed]) |
| `c.Drive()` | Transport handles (Registry[*DriveHandle]) |
| `c.Fs()` | Filesystem I/O (sandboxable) |
| `c.Process()` | Managed execution (Action sugar) |
| `c.API()` | Remote streams (protocol handlers) |
| `c.Action(name)` | Named callable (register/invoke) |
| `c.Task(name)` | Composed Action sequence |
| `c.Entitled(name)` | Permission check |
| `c.RegistryOf(n)` | Cross-cutting registry queries |
| `c.Cli()` | CLI command framework |
| `c.IPC()` | Message bus (ACTION, QUERY) |
| `c.Log()` | Structured logging |
| `c.Error()` | Panic recovery |
| `c.I18n()` | Internationalisation |

#### Primitive Types

```go
// Option — the atom
core.Option{Key: "name", Value: "brain"}

// Options — universal input
opts := core.NewOptions(
    core.Option{Key: "name", Value: "myapp"},
    core.Option{Key: "port", Value: 8080},
)
opts.String("name") // "myapp"
opts.Int("port")    // 8080

// Result — universal output
core.Result{Value: svc, OK: true}
```

#### Named Actions — The Primary Communication Pattern

Services register capabilities as named Actions. No direct function calls, no untyped dispatch — declare intent by name, invoke by name.

```go
// Register a capability during OnStartup
c.Action("workspace.create", func(ctx context.Context, opts core.Options) core.Result {
    name := opts.String("name")
    path := core.JoinPath("/srv/workspaces", name)
    return core.Result{Value: path, OK: true}
})

// Invoke by name — typed, inspectable, entitlement-checked
r := c.Action("workspace.create").Run(ctx, core.NewOptions(
    core.Option{Key: "name", Value: "alpha"},
))

// Check capability before calling
if c.Action("process.run").Exists() { /* go-process is registered */ }

// List all capabilities
c.Actions()  // ["workspace.create", "process.run", "brain.recall", ...]
```

#### Task Composition — Sequencing Actions

```go
c.Task("agent.completion", core.Task{
    Steps: []core.Step{
        {Action: "agentic.qa"},
        {Action: "agentic.auto-pr"},
        {Action: "agentic.verify"},
        {Action: "agentic.poke", Async: true},  // doesn't block
    },
})
```

#### Anonymous Broadcast — Legacy Layer

`ACTION` and `QUERY` remain for backwards-compatible anonymous dispatch. New code should prefer named Actions.

```go
// Broadcast — all handlers fire, type-switch to filter
c.ACTION(messages.DeployCompleted{Env: "production"})

// Query — first responder wins
r := c.QUERY(countQuery{})
```

#### Process Execution — Use Core Primitives

All external command execution MUST go through `c.Process()`, not raw `os/exec`. This makes process execution testable, gatable by entitlements, and managed by Core's lifecycle.

```go
// AX-native: Core Process primitive
r := c.Process().RunIn(ctx, repoDir, "git", "log", "--oneline", "-20")
if r.OK { output := r.Value.(string) }

// Not AX: raw exec.Command — untestable, no entitlement, no lifecycle
cmd := exec.Command("git", "log", "--oneline", "-20")
cmd.Dir = repoDir
out, err := cmd.Output()
```

**Rule:** If a package imports `os/exec`, it is bypassing Core's process primitive. The only package that should import `os/exec` is `go-process` itself.

**Quality gate:** An agent reviewing a diff can mechanically check: does this import `os/exec`, `unsafe`, or `encoding/json` directly? If so, it bypassed a Core primitive.

#### What This Replaces

| Go Convention | Core AX | Why |
|--------------|---------|-----|
| `func With*(v) Option` | `core.WithOption(k, v)` | Named key-value is greppable; option chains require tracing |
| `func Must*(v) T` | `core.Result` | No hidden panics; errors flow through Result.OK |
| `func *For[T](c) T` | `c.Service("name")` | String lookup is greppable; generics require type context |
| `val, err :=` everywhere | Single return via `core.Result` | Intent not obscured by error handling |
| `exec.Command(...)` | `c.Process().Run(ctx, cmd, args...)` | Testable, gatable, lifecycle-managed |
| `map[string]*T + mutex` | `core.Registry[T]` | Thread-safe, ordered, lockable, queryable |
| untyped `any` dispatch | `c.Action("name").Run(ctx, opts)` | Named, typed, inspectable, entitlement-checked |

### 7. Tests as Behavioural Specification

Test names are structured data. An agent querying "what happens when dispatch fails?" should find the answer by scanning test names, not reading prose.

```
TestDispatch_DetectFinalStatus_Good    — clean exit → completed
TestDispatch_DetectFinalStatus_Bad     — non-zero exit → failed
TestDispatch_DetectFinalStatus_Ugly    — BLOCKED.md overrides exit code
```

**Convention:** `Test{File}_{Function}_{Good|Bad|Ugly}`

| Category | Purpose |
|----------|---------|
| `_Good` | Happy path — proves the contract works |
| `_Bad` | Expected errors — proves error handling works |
| `_Ugly` | Edge cases, panics, corruption — proves it doesn't blow up |

**Rule:** Every testable function gets all three categories. Missing categories are gaps in the specification, detectable by scanning:

```bash
# Find under-tested functions
for f in *.go; do
  [[ "$f" == *_test.go ]] && continue
  while IFS= read -r line; do
    fn=$(echo "$line" | sed 's/func.*) //; s/(.*//; s/ .*//')
    [[ -z "$fn" || "$fn" == register* ]] && continue
    cap="${fn^}"
    grep -q "_${cap}_Good\|_${fn}_Good" *_test.go || echo "$f: $fn missing Good"
    grep -q "_${cap}_Bad\|_${fn}_Bad"   *_test.go || echo "$f: $fn missing Bad"
    grep -q "_${cap}_Ugly\|_${fn}_Ugly" *_test.go || echo "$f: $fn missing Ugly"
  done < <(grep "^func " "$f")
done
```

**Rationale:** The test suite IS the behavioural spec. `grep _TrackFailureRate_ *_test.go` returns three concrete scenarios — no prose needed. The naming convention makes the entire test suite machine-queryable. An agent dispatched to fix a function can read its tests to understand the full contract before making changes.

**What this replaces:**

| Convention | AX Test Naming | Why |
|-----------|---------------|-----|
| `TestFoo_works` | `TestFile_Foo_Good` | File prefix enables cross-file search |
| Unnamed table tests | Explicit Good/Bad/Ugly | Categories are scannable without reading test body |
| Coverage % as metric | Missing categories as metric | 100% coverage with only Good tests is a false signal |

### 7b. Example Tests as AX TDD

Go `Example` functions serve triple duty: they run as tests (count toward coverage), show in godoc (usage documentation), and seed user guide generation.

```go
// file: action_example_test.go

func ExampleAction_Run() {
    c := New()
    c.Action("double", func(_ context.Context, opts Options) Result {
        return Result{Value: opts.Int("n") * 2, OK: true}
    })

    r := c.Action("double").Run(context.Background(), NewOptions(
        Option{Key: "n", Value: 21},
    ))
    Println(r.Value)
    // Output: 42
}
```

**AX TDD pattern:** Write the Example first — it defines how the API should feel. If the Example is awkward, the API is wrong. The Example IS the test, the documentation, and the design feedback loop.

**Convention:** One `{source}_example_test.go` per source file. Every exported function should have at least one Example. The Example output comment makes it a verified test.

**Quality gate:** A source file without a corresponding example file is missing documentation that compiles.

### Operational Principles

Principles 1-7 govern code design. Principles 8-10 govern how agents and humans work with the codebase.

### 8. RFC as Domain Load

An agent's first action in a session should be loading the repo's RFC.md. The full spec in context produces zero-correction sessions — every decision aligns with the design because the design is loaded.

**Validated:** Loading core/go's RFC.md (42k tokens from a 500k token discovery session) at session start eliminated all course corrections. The spec is compressed domain knowledge that survives context compaction.

**Rule:** Every repo that has non-trivial architecture should have a `docs/RFC.md`. The RFC is not documentation for humans — it's a context document for agents. It should be loadable in one read and contain everything needed to make correct decisions.

### 9. Primitives as Quality Gates

Core primitives become mechanical code review rules. An agent reviewing a diff checks:

| Import | Violation | Use Instead |
|--------|-----------|-------------|
| `os` | Bypasses Fs/Env primitives | `c.Fs()`, `core.Env()`, `core.DirFS()`, `Fs.TempDir()` |
| `os/exec` | Bypasses Process primitive | `c.Process().Run()` |
| `io` | Bypasses stream primitives | `core.ReadAll()`, `core.WriteAll()`, `core.CloseStream()` |
| `fmt` | Bypasses string/print primitives | `core.Println()`, `core.Sprintf()`, `core.Sprint()` |
| `errors` | Bypasses error primitive | `core.NewError()`, `core.E()`, `core.Is()`, `core.As()` |
| `log` | Bypasses logging | `core.Info()`, `core.Warn()`, `core.Error()`, `c.Log()` |
| `encoding/json` | Bypasses Core serialisation | `core.JSONMarshal()`, `core.JSONUnmarshal()` |
| `path/filepath` | Bypasses path security boundary | `core.Path()`, `core.JoinPath()`, `core.PathBase()` |
| `unsafe` | Bypasses Fs sandbox | `Fs.NewUnrestricted()` |
| `strings` | Bypasses string guardrails | `core.Contains()`, `core.Split()`, `core.Trim()`, etc. |

**Rule:** If a diff introduces a disallowed import, it failed code review. The import list IS the quality gate. No subjective judgement needed — a weaker model can enforce this mechanically.

### 10. Registration IS Capability, Entitlement IS Permission

Two layers of permission, both declarative:

```
Registration = "this action EXISTS"     → c.Action("process.run").Exists()
Entitlement  = "this Core is ALLOWED"  → c.Entitled("process.run").Allowed
```

A sandboxed Core has no `process.run` registered — the action doesn't exist. A SaaS Core has it registered but entitlement-gated — the action exists but the workspace may not be allowed to use it.

**Rule:** Never check permissions with `if` statements in business logic. Register capabilities as Actions. Gate them with Entitlements. The framework enforces both — `Action.Run()` checks both before executing.

## Applying AX to Existing Patterns

### File Structure

```
# AX-native: path describes content
core/agent/
├── cmd/core-agent/          # CLI entry point (minimal — just core.New + Run)
├── pkg/agentic/             # Agent orchestration (dispatch, prep, verify, scan)
├── pkg/brain/               # OpenBrain integration
├── pkg/lib/                 # Embedded templates, personas, flows
├── pkg/messages/            # Typed IPC message definitions
├── pkg/monitor/             # Agent monitoring + notifications
├── pkg/setup/               # Workspace scaffolding + detection
└── claude/                  # Claude Code plugin definitions

# Not AX: generic names requiring README
src/
├── lib/
├── utils/
└── helpers/
```

### Error Handling

```go
// AX-native: errors flow through Result, not call sites
func Register(c *core.Core) core.Result {
    svc := &MyService{ServiceRuntime: core.NewServiceRuntime(c, MyOpts{})}
    return core.Result{Value: svc, OK: true}
}

// Not AX: errors dominate the code
func Register(c *core.Core) (*MyService, error) {
    svc, err := NewMyService(c)
    if err != nil {
        return nil, fmt.Errorf("create service: %w", err)
    }
    return svc, nil
}
```

### Command Registration

```go
// AX-native: extracted methods, testable without CLI
func (s *MyService) OnStartup(ctx context.Context) core.Result {
    c := s.Core()
    c.Command("issue/get", core.Command{Action: s.cmdIssueGet})
    c.Command("issue/list", core.Command{Action: s.cmdIssueList})
    c.Action("forge.issue.get", s.handleIssueGet)
    return core.Result{OK: true}
}

func (s *MyService) cmdIssueGet(opts core.Options) core.Result {
    // testable business logic — no closure, no CLI dependency
}

// Not AX: closures that can only be tested via CLI integration
c.Command("issue/get", core.Command{
    Action: func(opts core.Options) core.Result {
        // 50 lines of untestable inline logic
    },
})
```

### Process Execution

```go
// AX-native: Core Process primitive, testable with mock handler
func (s *MyService) getGitLog(repoPath string) string {
    r := s.Core().Process().RunIn(context.Background(), repoPath, "git", "log", "--oneline", "-20")
    if !r.OK { return "" }
    return core.Trim(r.Value.(string))
}

// Not AX: raw exec.Command — untestable, no entitlement check, path traversal risk
func (s *MyService) getGitLog(repoPath string) string {
    cmd := exec.Command("git", "log", "--oneline", "-20")
    cmd.Dir = repoPath  // user-controlled path goes directly to OS
    output, err := cmd.Output()
    if err != nil { return "" }
    return strings.TrimSpace(string(output))
}
```

The AX-native version routes through `c.Process()` → named Action → entitlement check. The non-AX version passes user input directly to `os/exec` with no permission gate.

### Permission Gating

```go
// AX-native: entitlement checked by framework, not by business logic
c.Action("agentic.dispatch", func(ctx context.Context, opts core.Options) core.Result {
    // Action.Run() already checked c.Entitled("agentic.dispatch")
    // If we're here, we're allowed. Just do the work.
    return dispatch(ctx, opts)
})

// Not AX: permission logic scattered through business code
func handleDispatch(ctx context.Context, opts core.Options) core.Result {
    if !isAdmin(ctx) && !hasPlan(ctx, "pro") {
        return core.Result{Value: core.E("dispatch", "upgrade required", nil), OK: false}
    }
    // duplicate permission check in every handler
}
```

## Compatibility

AX conventions are valid, idiomatic Go/PHP/TS. They do not require language extensions, code generation, or non-standard tooling. An AX-designed codebase compiles, tests, and deploys with standard toolchains.

The conventions diverge from community patterns (functional options, Must/For, etc.) but do not violate language specifications. This is a style choice, not a fork.

## Adoption

AX applies to all code in the Core ecosystem. core/go is fully migrated (v0.8.0). Consumer packages migrate via their RFCs.

Priority for migrating a package:
1. **Lifecycle** — `OnStartup`/`OnShutdown` return `Result`
2. **Actions** — register capabilities as named Actions
3. **Imports** — replace all 10 disallowed imports (Principle 9)
4. **String ops** — `+` concat → `Concat()`, `path +` → `Path()`
5. **Test naming** — `TestFile_Function_{Good,Bad,Ugly}`
6. **Examples** — one `{source}_example_test.go` per source file
7. **Comments** — every exported function has usage example (Principle 2)

## Verification

An agent auditing AX compliance checks:

```bash
# Disallowed imports (Principle 9)
grep -rn '"os"\|"os/exec"\|"io"\|"fmt"\|"errors"\|"log"\|"encoding/json"\|"path/filepath"\|"unsafe"\|"strings"' *.go \
  | grep -v _test.go

# Test naming (Principle 7)
grep "^func Test" *_test.go | grep -v "Test[A-Z][a-z]*_.*_\(Good\|Bad\|Ugly\)"

# String concat (should use Concat/Path)
grep -n '" + \| + "' *.go | grep -v _test.go | grep -v "//"

# Untyped dispatch (should prefer named Actions)
grep "RegisterTask\|PERFORM\|type Task any" *.go
```

If any check produces output, the code needs migration.

## References

- `core/go/docs/RFC.md` — CoreGO API contract (21 sections, reference implementation)
- `core/go-process/docs/RFC.md` — Process consumer spec
- `core/agent/docs/RFC.md` — Agent consumer spec
- RFC-004 (Entitlements) — permission model ported to `c.Entitled()`
- RFC-021 (Core Platform Architecture) — 7-layer stack, provider model
- dAppServer unified path convention (2024) — path = route = command = test
- Go Proverbs, Rob Pike (2015) — AX provides an updated lens

## Changelog

- 2026-03-25: v0.8.0 alignment — all examples match implemented API. Added Principles 8 (RFC as Domain Load), 9 (Primitives as Quality Gates), 10 (Registration + Entitlement). Updated subsystem table (Process, API, Action, Task, Entitled, RegistryOf). Process examples use `c.Process()` not old `process.RunWithOptions`. Removed PERFORM references.
- 2026-03-19: Initial draft — 7 principles
