# CODEX.md

Instructions for Codex when working in `dappco.re/go/agent`.

Read these files in order:
1. `CODEX.md`
2. `.core/reference/RFC-025-AGENT-EXPERIENCE.md`
3. `.core/reference/docs/RFC.md`
4. `AGENTS.md`

## Overview

This repo is the Core ecosystem's agent orchestration service. It is AX-first: predictable names, named Actions, Core primitives, and behaviour-driven tests matter more than terse APIs.

## Build And Test

```bash
go build ./...
go build ./cmd/core-agent/
go test ./... -count=1 -timeout 60s
go vet ./...
```

## Core Registration Pattern

Register services through `core.New` and `WithService`, not ad hoc globals.

```go
c := core.New(
    core.WithOption("name", "core-agent"),
    core.WithService(agentic.ProcessRegister),
    core.WithService(agentic.Register),
    core.WithService(runner.Register),
    core.WithService(monitor.Register),
    core.WithService(brain.Register),
)
c.Run()
```

## Mandatory Conventions

- Use UK English in comments and docs.
- Use `core.E("pkg.Method", "message", err)` for errors. Never use `fmt.Errorf` or `errors.New`.
- Use Core filesystem helpers or package-level `fs`. Never use raw `os.ReadFile`, `os.WriteFile`, or `filepath.*`.
- Route external commands through `pkg/agentic/proc.go` or `s.Core().Process()`. Never import `os/exec`.
- Use Core string helpers such as `core.Contains`, `core.Trim`, and `core.Split` instead of `strings.*`.
- Prefer `core.Result{Value: x, OK: true}` over `(value, error)` return pairs in Core-facing code.
- Comments should show real usage examples, not restate the signature.
- Prefer predictable names such as `Config`, `Service`, and `Options`; avoid abbreviations.
- Add `// SPDX-License-Identifier: EUPL-1.2` to Go source files.

## AX Quality Gates

Treat these imports as review failures in non-test Go code:

- `os`
- `os/exec`
- `fmt`
- `log`
- `errors`
- `encoding/json`
- `path/filepath`
- `strings`
- `unsafe`

Use the Core primitive or the repo helper instead.

## Testing

Use AX test naming:

```text
TestFile_Function_Good
TestFile_Function_Bad
TestFile_Function_Ugly
```

One source file should have its own focused test file and example file where practical. The test suite is the behavioural spec.

## Commits

Use `type(scope): description` and include:

```text
Co-Authored-By: Virgil <virgil@lethean.io>
```
