<!-- SPDX-License-Identifier: EUPL-1.2 -->
# DOCS-TASK.md — write core/agent feature docs from the code

> **Handoff brief for an autonomous agent.** Self-contained. Open this repo
> (`~/Code/core/agent`), read this file, then fill each `docs/<feature>/README.md`
> stub with **literal feature documentation written FROM THE CODE**.
>
> **Launch line** (paste into a window rooted at `~/Code/core/agent`):
> *"Read `DOCS-TASK.md` and execute it. Document each feature stub from the code,
> one commit per feature. Don't touch `plans/` or any `_test.go`."*

## Goal

`docs/` holds **only literal feature documentation** — what the code actually does,
in subfolders, one per feature. The stubs exist; fill them. When every stub is a
real doc with no `TODO` left, delete this file.

## Rules (non-negotiable)

- **From the code, not from memory.** Read the source for each feature; document
  what's there. Cite `file:Symbol` for entry points. If the code contradicts a
  belief, the code wins.
- **No specs/RFCs.** Those live in `plans/code/core/agent/` (the spec tree) — never
  duplicate them here. No roadmap, no promo, no "future work".
- **Literal + present-tense.** "X does Y" / "the `Foo` tool calls `Bar`". Describe
  behaviour, config flags (`auto-*`), MCP tools + CLI verbs, by-design gotchas.
- **Cross-link** `../known-issues.md` and sibling feature docs where relevant.
- **One commit per feature:** `docs(agent): document <feature> from code` with the
  exact trailer `Co-Authored-By: Virgil <virgil@lethean.io>`. UK English. EUPL-1.2.
- **Don't touch** `plans/`, `_test.go` files, or any Go source — this is docs only.

## The feature map (stub → code to read)

| stub | code |
|------|------|
| `docs/cli/` | `go/cmd/core-agent/` — `main.go`, `commands*.go`, `update.go` (modes: mcp, serve, chat, models, shell, update) |
| `docs/dispatch/` | `go/pkg/agentic/{dispatch,prep,resume,watch,queue,runtime}*.go` |
| `docs/pipeline/` | `go/pkg/agentic/{pipeline,qa,verify,*pr,merge,result,sanitise}*.go` + `go/pkg/messages/` |
| `docs/runner/` | `go/pkg/runner/` + `go/pkg/agentcompat/` |
| `docs/monitor/` | `go/pkg/monitor/` |
| `docs/fleet/` | `go/pkg/agentic/{fleet,platform,sync,register,repo}*.go` |
| `docs/remote/` | `go/pkg/agentic/remote*.go` |
| `docs/plans/` | `go/pkg/agentic/{plan,phase,session,sprint,state,statestore}*.go` |
| `docs/scan-mirror/` | `go/pkg/agentic/{scan,mirror,repo}*.go` |
| `docs/review/` | `go/pkg/agentic/review*.go` |
| `docs/opencode/` | `go/pkg/opencode/` + `go/pkg/agentic/opencode*.go` + `go/cmd/core-agent/commands_opencode.go` |
| `docs/shell/` | `go/pkg/agentic/shell*.go` + `go/cmd/core-agent/commands_shell.go` |
| `docs/lib/` | `go/pkg/lib/` (workspace, prompt, task, persona, flow) |
| `docs/content/` | `go/pkg/agentic/{content,training}*.go` |
| `docs/audit/` | `go/pkg/audit/` |

**Already written (verify against code, extend only if drifted):**
`docs/brain/callers.md` (`go/pkg/brain/`), `docs/inference/*` (`go/pkg/lemma/` + `go/pkg/chathistory/`), `docs/setup/github-app.md` (also document `go/pkg/setup/` workspace scaffolding here).

## Method (per feature)

1. Read the listed source files for the feature.
2. Write `Purpose` (what it does), `Entry points` (key funcs/types/tools/verbs with
   `file:Symbol` cites), `Behaviour` (the real flow + flags + gotchas).
3. Remove the stub banner + every `TODO`.
4. Verify each claim is traceable to the code you cited.
5. Commit (one per feature).

## Done

Every `docs/<feature>/README.md` is a real doc (no stub banner, no `TODO`), links
resolve, claims trace to code. The top-level `docs/` keeps only feature docs:
`architecture.md`, `development.md`, `known-issues.md`, + the feature subfolders.
Then delete `DOCS-TASK.md`.
