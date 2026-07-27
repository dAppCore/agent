# CLAUDE.md

This file provides guidance to Claude Code when working with code in this repository.

## Session Context

Running on **Claude Max20 plan** with **1M context window** (Opus 4.8).

## Overview

**core-agent** is the AI agent orchestration platform for the Core ecosystem. Single Go binary (`core-agent`) that runs as an MCP server — either via stdio (Claude Code integration) or HTTP daemon (cross-agent communication).

**Module:** `dappco.re/go/agent`

**Source of truth:** the RFC specs live in the plans tree at `plans/code/core/agent/` (`RFC.md`, `RFC.pipeline.md`, `RFC.topology.md`, `RFC.serve.md`, `flow/`, `plugins/`) — the present-tense contract for every subsystem. `docs/` in this repo holds literal feature documentation only — `architecture.md`, `known-issues.md`, a `development/` guide, and a folder per feature (each a URL: `dispatch/`, `pipeline/`, `plans/`, `brain/`, `inference/`, `providers/`, …) whose `README.md` is a concise SEO index linking to detail pages. This file is the operational quick-reference; when docs and code disagree, the code wins.

## Build & Test

```bash
go build ./...                        # Build all packages
go build ./cmd/core-agent/            # Build the binary
go test ./... -count=1 -timeout 60s   # Run tests
go vet ./...                          # Vet
go install ./cmd/core-agent/          # Install to $GOPATH/bin
```

Cross-compile for Charon (Linux):
```bash
GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -o core-agent-linux ./cmd/core-agent/
```

## Architecture

```
cmd/core-agent/main.go           Entry point — core.New + services + CLI run
pkg/agentic/                     MCP dispatch tools, IPC pipeline, plans/phases/sessions, fleet/platform sync
pkg/brain/                        OpenBrain — recall, remember, forget, list, messaging
pkg/lemma/                        Local lthn-mlx client — chat sessions + /v1/admin control
pkg/chathistory/                  Per-user portable DuckDB chat archive
pkg/monitor/                      Background monitoring + repo sync
pkg/runner/                       Local + container runners + dispatch queue
pkg/setup/                        Project detection + .core/ scaffolding
pkg/lib/                          Embedded personas, prompt + flow + workspace templates (go:embed)
pkg/messages/                     Typed IPC message definitions
```

> Also `pkg/opencode/` — the sandboxed opencode host (Service Start/Stop/Generate, profiles, reverse-proxy, hub control + audit): the AUI surface (RFC.md §6).

### Binary Modes

- `core-agent mcp` — stdio MCP server for Claude Code (registered by the `dappco.re/go/mcp` service)
- `core-agent serve` — HTTP MCP daemon (Charon, CI, cross-agent)
- `core-agent hub` — loopback control plane: `--http 127.0.0.1:9201` (bearer) + `--mcp-http 127.0.0.1:9202` (fail-closed MCP), fronting the opencode control/proxy groups + brain with a non-optional audit edge (RFC.md §2/§6)
- `core-agent chat --user=<id>` — REPL against the local lthn-mlx engine, auto-captured to the user's archive
- `core-agent serve-status` / `serve-reload` / `serve-profiles` — inspect / hot-swap the local model engine
- `core-agent models-download` / `models-job` — queue + poll Hugging Face model downloads

### MCP Tools (common subset — full action surface in `RFC.md`)

| Category | Tools |
|----------|-------|
| Dispatch | `agentic_dispatch`, `agentic_dispatch_remote`, `agentic_status`, `agentic_status_remote` |
| Workspace | `agentic_prep_workspace`, `agentic_resume`, `agentic_watch` |
| PR/Review | `agentic_create_pr`, `agentic_list_prs`, `agentic_create_epic`, `agentic_review_queue` |
| Mirror | `agentic_mirror` (Forge → GitHub sync) |
| Scan | `agentic_scan` (Forge issues) |
| Brain | `brain_recall`, `brain_remember`, `brain_forget` |
| Messaging | `agent_send`, `agent_inbox`, `agent_conversation` |
| Plans | `agentic_plan_create`, `agentic_plan_read`, `agentic_plan_update`, `agentic_plan_delete`, `agentic_plan_list` |
| Files | `file_read`, `file_write`, `file_edit`, `file_delete`, `file_rename`, `file_exists`, `dir_list`, `dir_create` |
| Language | `lang_detect`, `lang_list` |

### Agent Types

| Agent | Command | Use |
|-------|---------|-----|
| `claude:opus` | Claude Code | Complex coding, architecture |
| `claude:sonnet` | Claude Code | Standard tasks |
| `claude:haiku` | Claude Code | Quick/cheap tasks, discovery |
| `gemini` | Gemini CLI | Fast batch ops |
| `codex` | Codex CLI | Autonomous coding |
| `codex:review` | Codex review | Deep security analysis |
| `coderabbit` | CodeRabbit CLI | Code quality review |
| `opencode` | `opencode run` | Sandboxed agent routed to local/free-compute model profiles (RFC.md §6) |
| `local` | Codex + ollama bridge | Local OSS model via host `ollama` |

### Dispatch Flow

```
dispatch → agent works → closeout sequence (review → fix → simplify → re-review)
    → commit → auto PR → inline tests → pass → auto-merge on Forge
    → push to GitHub → CodeRabbit reviews → merge or dispatch fix agent
```

### Personas (pkg/lib/persona/)

Personas across many domains (ads, blockchain, code, design, devops, plan, product, sales, secops, smm, spatial, support, testing). Path = context, filename = lens.

### Templates (pkg/lib/prompt/, pkg/lib/task/, pkg/lib/flow/)

Prompt + task templates for different task types (`coding`, `conventions`, `security`, `verify`, code review, simplifier), plus per-language flow definitions in `pkg/lib/flow/` and YAML upgrade flows in `pkg/lib/flow/upgrade/`.

## Key Patterns

### Shared Paths (pkg/agentic/paths.go)

All paths use `CORE_WORKSPACE` env var, fallback `~/Code/.core`:
- `WorkspaceRoot()` — agent workspaces
- `CoreRoot()` — ecosystem config
- `PlansRoot()` — agent plans
- `AgentName()` — `AGENT_NAME` env or hostname detection
- `GitHubOrg()` — `GITHUB_ORG` env or "dAppCore"

### Error Handling

`coreerr.E("pkg.Method", "message", err)` from go-log. Always 3 args. NEVER `fmt.Errorf`.

### File I/O

`coreio.Local.Read/Write/EnsureDir` from go-io. `WriteMode(path, content, 0600)` for sensitive files. NEVER `os.ReadFile/WriteFile`.

### HTTP Responses

Always check `err != nil` BEFORE accessing `resp.StatusCode`. Split into two checks.

## Plugin Providers (provider/)

core-agent ships its capabilities to a coding-agent host through two providers, one capability set (RFC.md §7):

- **`provider/claude/`** — Claude Code plugin: MCP server (`mcp.json`, auto-registers core-agent), hooks (`hooks.json` — inbox notifications, auto-format, debug warnings), agents (`agent-task-code-review`, `agent-task-code-simplifier`), commands (dispatch, status, review, recall, remember, scan…), skills (security / architecture / test review…).
- **`provider/opencode/`** — opencode plugin (`@opencode-ai/plugin`): capabilities as custom `tool()` exports (dispatch, status, scan, brain_recall…); `session.*` event hooks feeding the report-home loop; the ctx `client` SDK drives the running session. Personas ≡ opencode agent-defs (markdown frontmatter); skills ≡ `SKILL.md`; dispatch is two-layer (opencode `Task` subagents + core-agent's cross-host fleet), or attach the hub MCP plane via `POST /mcp`.

## Testing Conventions

- `_Good` — happy path
- `_Bad` — expected error conditions
- `_Ugly` — panics and edge cases
- Use `testify/assert` + `testify/require`

## Sprint Intel Collection

Before starting significant work on any repo, build a blueprint by querying three sources in parallel:

1. **OpenBrain**: `brain_recall` with `"{repo} plans features ideas architecture"` — returns bugs, patterns, conventions, session milestones
2. **Active plans**: `agentic_plan_list` — structured plans with phases, status, acceptance criteria
3. **Local docs**: glob `docs/plans/**` in the repo — design docs, migration plans, pipeline docs

Combine into a sprint blueprint with sections: Known Bugs, Active Plans, Local Docs, Recent Fixes, Architecture Notes.

### Active Plan: Pipeline Orchestration (draft)

Plans drive the entire dispatch→verify→merge flow:

1. **Plans API** — local JSON → CorePHP Laravel endpoints
2. **Plan ↔ Dispatch** — auto-advance phases, auto-create Forge issues on BLOCKED
3. **Task minting** — `/v1/plans/next` serves highest-priority ready phase
4. **Exception pipeline** — BLOCKED → Forge issues automatically
5. **GitHub quality gate** — verified → squash release, CodeRabbit 0-findings
6. **Pipeline dashboard** — admin UI with status badges

### Known Gotchas (OpenBrain)

- Workspace prep: PROMPT.md requires TODO.md but workspace may not have one — dispatch bug
- `core.Env("DIR_HOME")` is static at init. Use `CORE_HOME` for test overrides
- `pkg/brain` recall/list are async bridge proxies — empty responses are intentional
- Monitor path helpers need separator normalisation for cross-platform API/glob output

## Coding Standards

- **UK English**: colour, organisation, centre, initialise
- **Commits**: `type(scope): description` with `Co-Authored-By: Virgil <virgil@lethean.io>`
- **Licence**: EUPL-1.2
- **SPDX**: `// SPDX-License-Identifier: EUPL-1.2` on every file

## Repo Layout

```
core/agent/
├── go/                               ← Go module root (module dappco.re/go/agent)
│   ├── go.mod, go.sum
│   ├── cmd/                          ← Go binaries and entrypoints
│   ├── pkg/                          ← Go runtime packages
│   ├── version.go
│   ├── version_example_test.go
│   ├── README.md                     ← symlink → ../README.md
│   ├── CLAUDE.md                     ← symlink → ../CLAUDE.md
│   ├── AGENTS.md                     ← symlink → ../AGENTS.md
│   └── docs                          ← symlink → ../docs
├── php/                              ← PHP package (unchanged)
├── tests/                            ← repo test tooling/assets
├── scripts/                          ← task and maintenance helpers
├── docs/                             ← shared documentation
└── <other root files>                ← CI/config and PHP project files
```

## Go Resolution Modes

The Go module is located at `go/`, so run Go tooling from there:

- Development/default: `cd go && go build ./...`, `cd go && go test ./...`
- CI and explicit reproducibility checks: `GOWORK=off` (and optional `GOFLAGS=-mod=mod`) when running `go test`, `go vet`, and `go mod tidy` from `go/`.
