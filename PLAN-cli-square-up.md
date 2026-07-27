<!-- SPDX-License-Identifier: EUPL-1.2 -->
# Plan — square up core-agent's CLI + IPC handlers

> **Scope: core/agent only.** core/agent consumes `dappco.re/go/cli` and `dappco.re/go` as
> services. core/cli internals and other repos' migrations are out of lane — referenced as
> dependencies, never owned or rewritten from here.

## Read this first — the mental model the last attempt got wrong

A previous agent read this plan's old Phase 2 ("mount the actions onto the CLI as commands") and went
sideways: it stood up a **second `*core.Cli`**, wrote a core/agent **`action_mount.go`**, and bolted on
work-arounds — instead of reading how the pieces already fit. That work was reverted. The truth:

- **The CLI is already up, via the service.** `core.WithService(cli.Register)` registers the `*core.Cli`
  primitive (`core.CliRegister`) + the `cli.*` diagnostic actions. `c.Cli()` resolves; `Core.Run()` drives
  it (`ServiceStartup → cli.Run() → ServiceShutdown`). `version` / `check` work. **Build is green.** There
  is nothing left to "wire" for the CLI to exist — it composes like any other service.
- **Actions are the capability map, surfaced over the *bus*, not the CLI.** The ~228 actions
  (`runner.dispatch`, `agentic.qa`, …) are reachable via **IPC** (`c.ACTION(…)`, `c.Action("x.y").Run(…)`,
  `c.Query(…)`), via **MCP** (`coremcp.Register` projects them as tools), and via the **hub** HTTP plane.
  That is what "already mounted via the service" means. **Verified:** `core-agent runner status` does *not*
  resolve as a CLI command (it falls through to help) — and that is **correct**. We do **not** want 228 CLI
  subcommands; the CLI carries only the explicit human-facing commands (the 13 in `cmd/core-agent/commands.go`).

### Anti-patterns — do NOT do these (each is a reverted dead-end)
- ❌ **No second `*core.Cli`.** No `cli.Init` / `cli.Main` / `cli.Execute` in `main.go`. The cli is the one
  `cli.Register` stood up. A second one double-registers and panics.
- ❌ **No `cli.MountActions`, no core/agent `action_mount.go`.** `MountActions` is a core/**cli** *library*
  primitive for binaries that deliberately want every action as a CLI subcommand. core/agent is **not** one
  of those — its actions surface over IPC/MCP/hub. Do not call it; do not reimplement it; do not touch
  `external/cli/.../action_mount.go`.
- ❌ **No hand-wiring actions as commands.** If you find yourself adding `c.Command("runner/status", …)` to
  expose an action, stop — that action is already reachable on the bus.
- ✅ **The only pattern here:** a `messages.X` event is emitted with `c.ACTION(messages.X{…})`; a
  **handler** reacts to it (re-dispatches to an action / sends a notification / updates state). The work is
  **adding handlers**. Nothing else.

## Phase 1 — CLI on the service — DONE
`core.WithService(cli.Register)` + `Core.Run()`. Remaining housekeeping (one commit on `dev`):
- Collapse `runApp` in `cmd/core-agent/main.go` (`ServiceStartup` + `coreApp.Cli().Run()` + `ServiceShutdown`)
  to `coreApp.Run()` — *after* the binary-name banner/name override is set (`Core.Run()` takes no args; it
  reads argv itself, same as the current `startupArgs()` path used by `cli.Run`).
- Commit core/agent: `go.work` (submodule-only, zero `../` refs — already true), `main.go`, and the submodule
  bumps (external/go, external/cli, external/orm, external/go-container).
- **Done when:** `version` / `check` green; `go test ./...`; one clean commit on `dev`.

## Phase 2 — the actual work: IPC handlers for emitted-but-unhandled events
These five `messages.*` events are **emitted for real** and **handled by nobody** — broadcast to the floor.
Each needs a handler. (Instrument: `grep -rn '\.ACTION(messages\.X' pkg/` for emits; `grep -rn 'case messages.X\|(messages.X)' pkg/` for handlers.)

| # | event (payload) | emitted at | proposed reaction | host |
|---|---|---|---|---|
| H1 | `QueueDrained{Completed int}` | monitor.go:248,406 · runner.go:423 | notify the mcp status channel that the queue drained (`Completed`); the queue lifecycle is now observable | a `case` in `runner.HandleIPCEvents` (runner.go:124 — already has `sendNotification`) |
| H2 | `RateLimitDetected{Pool, Duration}` | dispatch.go:557 | notify; **decide:** also back off that pool's dispatch for `Duration` (runner has only a *global* `frozen` flag today — per-pool backoff is new logic; notify-only is a valid v1) | `runner.HandleIPCEvents` |
| H3 | `HarvestComplete{Repo, Branch, Files}` | harvest.go:51 | notify the harvest channel (`Files` harvested); **decide:** whether to also re-dispatch `agentic.auto-pr`/`agentic.commit` for the harvested branch (ties to task #96) | `runner.HandleIPCEvents` or a `RegisterActions` handler in `agentic` |
| H4 | `HarvestRejected{Repo, Branch, Reason}` | harvest.go:46 | notify the harvest channel with `Reason` so a rejected harvest is visible, not silent | same as H3 |
| H5 | `InboxMessage{New, Total}` | monitor.go:493 · agentic/message.go:98 | notify the inbox/status channel (`New`/`Total`) so OpenBrain inbox arrivals surface (ties to task #218) | `runner.HandleIPCEvents` |

**decide:** tags are real choices for the implementer to confirm with Snider — do not invent rich backoff /
auto-PR behaviour unprompted. The safe, always-correct floor for all five is **notify** (it ports the
existing `AgentStarted` notification path); the richer reactions (H2 backoff, H3 auto-PR) are opt-in.

## The canonical pattern — copy this, do not improvise
Two equivalent ways to add a handler; both are already in the tree — read them before writing:

**A. A `case` in a service's `HandleIPCEvents`** (the message-bus reaction; auto-wired by `RegisterService`).
`runner.HandleIPCEvents` (runner.go:124) is the model — it already type-switches and calls a local
`sendNotification(channel, data)` that resolves the `mcp` service and `ChannelSend`s:
```go
case messages.QueueDrained:           // H1
    sendNotification("queue.status", &QueueNotification{Completed: ev.Completed})
```

**B. A standalone handler registered in the service's `Register`** via `c.RegisterActions(…)` — the model is
`agentic/handlers.go:15` (`RegisterHandlers`), where each handler type-asserts and **re-dispatches to an
action**:
```go
func handleHarvestComplete(c *core.Core, msg core.Message) core.Result {
    ev, ok := msg.(messages.HarvestComplete)
    if !ok { return core.Result{OK: true} }   // not our event — pass
    // re-dispatch (don't wire): performAsyncIfRegistered(c, "agentic.auto-pr", …)  // decide: H3
    return core.Result{OK: true}
}
```
Re-dispatch verbs already in use: `c.Action("x.y").Run(ctx, opts)` (sync), `c.PerformAsync("x.y", opts)`
(async; see `performAsyncIfRegistered`), `c.ACTION(messages.Y{…})` (chain another event). A handler that
doesn't recognise the message **must** return `core.Result{OK: true}` — broadcast hits every handler.

The event vocabulary is `pkg/messages/messages.go` (16 DTOs). Need a new event? Add a DTO there first.

## Done-when (per handler) + tests (AX-10)
Each handler ships with a test that **emits the event and asserts the reaction** — the established shape in
`pkg/agentic/handlers_test.go` / `pkg/runner/*_test.go`: build a `core.New(...)` with the service, call
`c.ACTION(messages.X{…})` (or the handler directly), assert the side effect (channel notified / action
dispatched / state changed). Plus `{file}_test.go` + `{file}_example_test.go` for any new file.

## Dependencies (consumed, not owned here)
- **core/cli** — already provides `cli.Register` (the cli service) + `action_mount.go` (the lib primitive we
  *don't* use). No core/cli change is needed for this plan.
- **core/go** IPC surface — `c.ACTION` (broadcast), `RegisterAction`/`RegisterActions`, `HandleIPCEvents`
  auto-discovery via `RegisterService` (service.go:113). The mcp service supplies `ChannelSend`
  (the `channelSender` interface runner already uses).

## Conventions
Errors via `core.E(...)`; UK English; `// SPDX-License-Identifier: EUPL-1.2` on every file; each `{file}.go`
ships `{file}_test.go` + `{file}_example_test.go`. Push forge→github `dev`, non-force; bump submodules after
dependency changes. Commit trailer `Co-Authored-By: Virgil <virgil@lethean.io>`.

## Status (2026-06-27)
- **H1 / H4 / H5 landed** — notify cases in `runner.HandleIPCEvents` (`queue.status` / `harvest.status` /
  `inbox.status`) + typed payloads + tests (`runner_ipc_handlers_test.go`). Green.
- **H2 landed** — Snider's call: **back off + notify**. `RateLimitDetected` writes the runner's per-pool
  `backoff` map (under the same `runner.drain` lock `drainOne` reads it through, so no map-race) → the pool
  pauses for `Duration`; surfaced on `ratelimit.status`. The backoff map was read at `queue.go:219` but had
  no writer until this handler. Tests cover the backoff + the malformed-duration (notify, no freeze) path.
- **H3 landed** — Snider's call: **re-dispatch auto-PR + notify**. `runner` notifies `harvest.status`;
  `agentic.handleHarvestAutoPR` (registered in `RegisterHandlers`) re-dispatches `agentic.auto-pr` for the
  harvested branch's workspace via `performAsyncIfRegistered`. Tests cover the redispatch + no-workspace no-op.
- **Phase 1 housekeeping** (collapse `runApp`→`coreApp.Run()` + the submodule-bump commit) still pending.
- **Pre-existing failure, NOT from this work:** `TestCommandsCore_CliHelp_Good_ListsAllSubcommands` fails on
  the clean tree (`captureStdout` → `signal: broken pipe`) — confirmed by stash-isolation. Possibly Phase 1
  (cli-on-service) fallout; needs its own look. Nothing else in runner/agentic regressed.
