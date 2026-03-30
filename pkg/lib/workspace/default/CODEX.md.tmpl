# CODEX.md

Instructions for Codex when working in this workspace.

Read these files in order:
1. `CODEX.md`
2. `.core/reference/RFC-025-AGENT-EXPERIENCE.md`
3. `.core/reference/docs/RFC.md`
4. `AGENTS.md` if present

## Overview

This workspace follows RFC-025 Agent Experience (AX) design. Prefer predictable names, named Actions, Core primitives, usage-example comments, and behavioural tests over terse APIs. Use `.core/reference/*.go` as the local implementation reference.

## Core Registration Pattern

Register services through `core.New` and `WithService`, not ad hoc globals.

```go
c := core.New(
    core.WithOption("name", "my-service"),
    core.WithService(myservice.Register),
)
c.Run()
```

## Service Pattern

Services should be Result-native and register capabilities by name.

```go
func Register(c *core.Core) core.Result {
    svc := &Service{
        ServiceRuntime: core.NewServiceRuntime(c, Options{}),
    }
    return core.Result{Value: svc, OK: true}
}

func (s *Service) OnStartup(ctx context.Context) core.Result {
    c := s.Core()

    c.Action("workspace.create", s.handleWorkspaceCreate)
    c.Task("workspace.bootstrap", core.Task{
        Steps: []core.Step{
            {Action: "workspace.create"},
        },
    })

    return core.Result{OK: true}
}

func (s *Service) OnShutdown(ctx context.Context) core.Result {
    return core.Result{OK: true}
}
```

## Core Accessors

| Accessor | Purpose |
|----------|---------|
| `c.Options()` | Input configuration |
| `c.Config()` | Runtime settings and feature flags |
| `c.Data()` | Embedded assets |
| `c.Fs()` | Filesystem I/O |
| `c.Process()` | Managed process execution |
| `c.API()` | Remote streams and protocol handles |
| `c.Action(name)` | Named callable registration and invocation |
| `c.Task(name)` | Composed Action sequence |
| `c.Entitled(name)` | Permission check |
| `c.RegistryOf(name)` | Cross-cutting registry lookup |
| `c.Cli()` | CLI command framework |
| `c.IPC()` | Message bus |
| `c.Log()` | Structured logging |
| `c.Error()` | Panic recovery |
| `c.I18n()` | Internationalisation |

## Named Actions And Tasks

Actions are the primary communication pattern. Register by name, invoke by name.

```go
c.Action("workspace.create", func(ctx context.Context, opts core.Options) core.Result {
    name := opts.String("name")
    path := core.JoinPath("/srv/workspaces", name)
    return core.Result{Value: path, OK: true}
})

r := c.Action("workspace.create").Run(ctx, core.NewOptions(
    core.Option{Key: "name", Value: "alpha"},
))
```

Use Tasks to compose orchestration declaratively.

```go
c.Task("deploy", core.Task{
    Steps: []core.Step{
        {Action: "docker.build"},
        {Action: "docker.push"},
        {Action: "deploy.ansible", Async: true},
    },
})
```

## Core Primitives

Route recurring concerns through Core primitives instead of the raw standard library.

```go
var fs = (&core.Fs{}).NewUnrestricted()

statusPath := core.JoinPath("/srv/workspaces", "alpha", "status.json")
read := fs.Read(statusPath)

run := c.Process().RunIn(ctx, repoDir, "git", "log", "--oneline", "-20")
if run.OK {
    output := core.Trim(run.Value.(string))
    core.Print(nil, output)
}
```

## Mandatory Conventions

- Use UK English in comments and docs.
- Use `core.E("pkg.Method", "message", err)` for errors. Never use `fmt.Errorf` or `errors.New`.
- Use `c.Fs()` or a package-level `fs` helper for file I/O. Never use raw `os.ReadFile`, `os.WriteFile`, or `filepath.*`.
- Route external commands through `c.Process()`. Never import `os/exec`.
- Use Core string and path helpers such as `core.Contains`, `core.Trim`, `core.Split`, `core.Concat`, and `core.JoinPath` instead of raw `strings.*` or path concatenation.
- Prefer `core.Result{Value: x, OK: true}` over `(value, error)` pairs in Core-facing code.
- Comments should show HOW with real values, not restate the signature.
- Use predictable names such as `Config`, `Service`, and `Options`; avoid abbreviations.
- Keep paths self-describing. Directory names should tell an agent what they contain.
- Prefer templates for recurring generated files and config shapes.

## AX Quality Gates

Treat these imports as review failures in production Go code:

- `os`
- `os/exec`
- `fmt`
- `log`
- `errors`
- `encoding/json`
- `path/filepath`
- `strings`
- `unsafe`

## Testing

Use AX test naming and keep example coverage close to the source.

```text
TestFile_Function_Good
TestFile_Function_Bad
TestFile_Function_Ugly
```

Where practical, keep one focused `_example_test.go` alongside each source file so the tests double as usage documentation.

## Build & Test

```bash
go build ./...
go test ./... -count=1 -timeout 60s
go vet ./...
```
