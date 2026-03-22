# CLAUDE.md

This file provides guidance to Claude Code when working with code in this repository.

## Session Context

Running on **Claude Max20 plan** with **1M context window** (Opus 4.6).

## Overview

**core-agent** is the AI agent orchestration platform for the Core ecosystem. Single Go binary (`core-agent`) that runs as an MCP server — either via stdio (Claude Code integration) or HTTP daemon (cross-agent communication).

**Module:** `forge.lthn.ai/core/agent`

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
cmd/core-agent/main.go          Entry point (mcp + serve commands)
pkg/agentic/                     MCP tools — dispatch, verify, remote, mirror, review queue
pkg/brain/                       OpenBrain — recall, remember, messaging
pkg/monitor/                     Background monitoring + repo sync
pkg/prompts/                     Embedded templates + personas (go:embed)
```

### Binary Modes

- `core-agent mcp` — stdio MCP server for Claude Code
- `core-agent serve` — HTTP daemon (Charon, CI, cross-agent). PID file, health check, registry.

### MCP Tools (33)

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

### Dispatch Flow

```
dispatch → agent works → closeout sequence (review → fix → simplify → re-review)
    → commit → auto PR → inline tests → pass → auto-merge on Forge
    → push to GitHub → CodeRabbit reviews → merge or dispatch fix agent
```

### Personas (pkg/prompts/lib/personas/)

116 personas across 16 domains. Path = context, filename = lens.

```
prompts.Persona("engineering/security-developer")   # code-level security review
prompts.Persona("smm/security-secops")              # social media incident response
prompts.Persona("devops/senior")                     # infrastructure architecture
```

### Templates (pkg/prompts/lib/templates/)

Prompt templates for different task types: `coding`, `conventions`, `security`, `verify`, plus YAML plan templates (`bug-fix`, `code-review`, `new-feature`, `refactor`, etc.)

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

## Plugin (claude/core/)

The Claude Code plugin provides:
- **MCP server** via `mcp.json` (auto-registers core-agent)
- **Hooks** via `hooks.json` (PostToolUse inbox notifications, auto-format, debug warnings)
- **Agents**: `agent-task-code-review`, `agent-task-code-simplifier`
- **Commands**: dispatch, status, review, recall, remember, scan, etc.
- **Skills**: security review, architecture review, test analysis, etc.

## Testing Conventions

- `_Good` — happy path
- `_Bad` — expected error conditions
- `_Ugly` — panics and edge cases
- Use `testify/assert` + `testify/require`

## Coding Standards

- **UK English**: colour, organisation, centre, initialise
- **Commits**: `type(scope): description` with `Co-Authored-By: Virgil <virgil@lethean.io>`
- **Licence**: EUPL-1.2
- **SPDX**: `// SPDX-License-Identifier: EUPL-1.2` on every file
