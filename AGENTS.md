# AGENTS.md — Universal Agent Instructions

> For all AI agents working on this repository (Codex, Claude, Gemini, etc).
> See also: `llm.txt` for entry points, RFC-025 for design conventions.

## Overview

**core-agent** is the AI agent orchestration platform for the Core ecosystem. Single Go binary that runs as an MCP server — stdio (IDE integration) or HTTP daemon (cross-agent communication).

**Module:** `dappco.re/go/agent`

## Build & Test

```bash
go build ./...                        # Build all packages
go build ./cmd/core-agent/            # Build the binary
go test ./... -count=1 -timeout 60s   # Run all tests (840+)
go vet ./...                          # Vet
```

## Architecture

```
cmd/core-agent/main.go    Entry point (97 lines — core.New + services + Run)
pkg/agentic/               Agent orchestration: dispatch, prep, verify, scan, review
pkg/brain/                  OpenBrain memory integration
pkg/lib/                    Embedded templates, personas, flows, workspace scaffolds
pkg/messages/               Typed IPC message definitions (12 message types)
pkg/monitor/                Agent monitoring, notifications, completion tracking
pkg/setup/                  Workspace detection and scaffolding
```

### Service Registration

```go
c := core.New(
    core.WithOption("name", "core-agent"),
    core.WithService(agentic.ProcessRegister),
    core.WithService(agentic.Register),
    core.WithService(monitor.Register),
    core.WithService(brain.Register),
    core.WithService(mcp.Register),
)
c.Run()
```

### Dispatch Flow

```
dispatch → prep workspace → spawn agent in Docker container
    → agent works → completion detected via proc.Done()
    → detectFinalStatus (completed/blocked/failed)
    → onAgentComplete (save output, track rate limits, emit IPC)
    → IPC pipeline: AgentCompleted → QA → AutoPR → Verify → Merge
```

## Coding Standards

- **UK English**: colour, organisation, centre, initialise
- **Errors**: `core.E("pkg.Method", "message", err)` — NEVER `fmt.Errorf`
- **File I/O**: Package-level `fs` (go-io Medium) — NEVER `os.ReadFile/WriteFile`
- **Processes**: `proc.go` helpers (go-process) — NEVER `os/exec` directly
- **Strings**: `core.Contains/Split/Trim/HasPrefix/Sprintf` — NEVER `strings.*`
- **Returns**: `core.Result{Value, OK}` — NEVER `(value, error)` pairs
- **Comments**: Usage examples showing HOW with real values, not descriptions
- **Names**: Predictable, unabbreviated (Config not Cfg, Service not Srv)
- **Commits**: `type(scope): description` with `Co-Authored-By: Virgil <virgil@lethean.io>`
- **Licence**: EUPL-1.2 — `// SPDX-License-Identifier: EUPL-1.2` on every file

## Testing Convention (AX-7)

Every function gets three test categories:

```
TestFilename_FunctionName_Good    — happy path, proves contract works
TestFilename_FunctionName_Bad     — expected errors, proves error handling
TestFilename_FunctionName_Ugly    — edge cases, panics, corruption
```

One test file per source file. No catch-all files. Names must sort cleanly.

### Current Coverage

- 840 tests, 79.9% statement coverage
- 92% AX-7 (Good/Bad/Ugly) category coverage

## Process Execution

All external commands go through `pkg/agentic/proc.go` → go-process:

```go
out, err := runCmd(ctx, dir, "git", "log", "--oneline")
ok := gitCmdOK(ctx, dir, "fetch", "origin", "main")
branch := gitOutput(ctx, dir, "rev-parse", "--abbrev-ref", "HEAD")
```

**NEVER import `os/exec`.** Zero source files do.

## MCP Tools (33)

| Category | Tools |
|----------|-------|
| Dispatch | `agentic_dispatch`, `agentic_dispatch_remote`, `agentic_status`, `agentic_status_remote` |
| Workspace | `agentic_prep_workspace`, `agentic_resume`, `agentic_watch` |
| PR/Review | `agentic_create_pr`, `agentic_list_prs`, `agentic_create_epic`, `agentic_review_queue` |
| Mirror | `agentic_mirror` (Forge → GitHub sync) |
| Scan | `agentic_scan` (Forge issues) |
| Brain | `brain_recall`, `brain_remember`, `brain_forget` |
| Messaging | `agent_send`, `agent_inbox`, `agent_conversation` |
| Plans | `agentic_plan_create/read/update/delete/list` |
| Files | `file_read/write/edit/delete/rename/exists`, `dir_list/create` |
| Language | `lang_detect`, `lang_list` |

## Key Paths

| Function | Returns |
|----------|---------|
| `WorkspaceRoot()` | `$CORE_WORKSPACE/workspace` or `~/Code/.core/workspace` |
| `CoreRoot()` | `$CORE_WORKSPACE` or `~/Code/.core` |
| `PlansRoot()` | `$CORE_WORKSPACE/plans` |
| `AgentName()` | `$AGENT_NAME` or hostname-based detection |
| `GitHubOrg()` | `$GITHUB_ORG` or `dAppCore` |

## Branch Strategy

- Work on `dev` branch, never push to `main` directly
- PRs required for `main`
- Use `go get -u ./...` for dependency updates, never manual go.mod edits
