<!-- SPDX-Licence-Identifier: EUPL-1.2 -->

# core/agent — RFC↔code Parity Survey

> Survey + verify-first spot-checks, **2026-05-31**, against `RFC.md` (415 lines, 18 §) and the Go
> module at `go/`. **Build / vet / test: GREEN** (14 packages `ok`, 0 vet findings).
>
> Method: a **survey** (locate each described behaviour; present/partial/missing; dependencies),
> then targeted **verify-first** reads that corrected several first-pass over-calls. The
> exhaustive forward+backward reconcile is the GOAL.md loop's job, run per unit during execution.

## Headline

The RFC tracks the code closely — it reads as if written *from* the code. **Forward parity is
HIGH across ~17 of 18 sections.** This is a **reconcile-dominated drive**, not a build-out:

1. **One clear forward-code item:** §12 report-home loop (RFC-acknowledged "out of action").
2. **Two verify-and-close items:** §6.5 `prompt_async`/proxy path coverage; §7 `provider/opencode`
   (appears absent/relocated).
3. **The bulk of the work is backward reconcile** — fold real, intended code behaviour that the
   RFC omits into `RFC.md` (§9 extra verbs, §15 extra config, §14 per-flow tools, command
   aliases, etc.), section by section, until a full pass finds zero gaps both ways.

## Verify-first corrections (first-pass over-calls, now resolved)

| First-pass claim | Reality (verified) |
|---|---|
| §10 phase/task verbs absent | ✅ present — `commands_phase.go` (`phase/get`, `phase/update_status`, `phase/add_checkpoint` + aliases), `commands_task.go` (`task/create`, `task/update`, `task/toggle`) |
| §11 fleet "depth unverified / maybe missing" | ✅ substantially present — `sync.go` (`/v1/agent/sync` push, `/v1/agent/context` pull, `syncBackoffSchedule`), `remote_sync_queue.go` (offline queue), `fleet_connect.go` (poll fallback), `auth.go`/`fleet_login.go` (pairing) |
| §3 models maybe PHP-only | ✅ all in Go — `plan.go`, `phase.go`, `session.go`, `message.go`, `auth.go` (AgentApiKey), `issue.go`, `sprint.go`, `prompt_version.go`, `template.go`, `state.go`, `brain/tools.go` (BrainMemory), `opencode/types.go` (Sandbox) |
| §13 content "no Go surface" | ✅ present & rich — `content.go` (931L): `content.generate`, `content.batch.generate`, `content.brief.{create,get,list}`, schema |
| §7 `provider/opencode` (first-pass said "exists", from a glitchy `ls`) | ⚠️ clean `ls provider/` shows `claude,codex,google,hermes` only — `provider/opencode` appears ABSENT; U9 verifies (relocated per Mantis #1807, or a real gap) |

## Real forward gaps (need code)

- **[high] §12 report-home loop** — emit side exists (`message.go:98` emits `messages.InboxMessage`,
  `monitor.go:493` likewise; `message.go:166` uses `ChannelInboxMessage`), but RFC §12
  self-acknowledges the live push-listener → plugin-surface loop is "currently out of action."
  Investigate the exact break and restore. **HEADLINE — the one clear build item.**
- **[low–med] §6.5 `prompt_async` / proxy coverage** — core-agent's own client (`generate.go`
  `Generate`) is sync-only (`/session` + `/session/:id/message`); `prompt_async` is reachable
  only if the proxy forwards the `/session` prefix (`proxy.go`). Verify the proxy covers the full
  §6.5 surface (`prompt_async`, `/children`, `/abort`, `/fork`, `/permissions`, `POST /mcp`,
  `/agent`, `/command`, `/global/health`); close any uncovered path. Decide if the fleet needs a
  typed async client.
- **[med] §7 `provider/opencode`** — clean survey shows `provider/{claude,codex,google,hermes}`
  only; the RFC's opencode plugin (`@opencode-ai/plugin`) appears absent or relocated (Mantis
  #1807). U9 verifies → reconcile, re-point, or build/correct §7.

## Backward gaps (code does more than RFC — fold into RFC)

- **§9 Forge**: `issue/assign`, `issue/report`, `repo/get`, `repo/list`, `repo/sync`,
  `plan/from-issue` (RFC §9 lists fewer).
- **§15 Config**: `pools`, `default_persona`, `personas`, `host_mounts` (`runner.go`/`queue.go`).
- **§14 / §2**: each flow auto-registers as its own MCP tool (`flow_tools.go`, Mantis #1806);
  nested flow composition with cycle+depth guards (Mantis #1805); `run/flow` + `agentic:run/flow`.
- **§10**: `plan/from-issue`, `plan/templates`, `plan/check`, status aliases.
- **command aliasing**: most verbs are double-registered bare + `agentic:`-prefixed — document the
  convention once in the RFC.
- (more expected during per-section reconcile — this is the survey, not the audit.)

## Per-section survey (corrected)

| § | Subsystem | Forward | Notes |
|---|-----------|---------|-------|
| 2 | Binary & modes | ✅ high | 11 verbs wired; `mcp`/`serve` via external `coremcp.Register` |
| 3 | Domain model | ✅ high | all types in Go (see corrections table) |
| 4 | Dispatch & workspace | ✅ high | `prep.go`/`dispatch.go`/`prompt.go`/`agent_command.go`/`container.go`; reconcile detail per-unit |
| 5 | Completion pipeline | ✅ high | 6-step chain + Poindexter + `.meta/report.json` present |
| 6 | opencode surface | 🟡 high | lifecycle/profiles/generate/hub present; verify `prompt_async`/proxy coverage |
| 7 | Plugin providers | 🟡 | `provider/{claude,codex,google,hermes}`; `provider/opencode` appears ABSENT — verify (U9) |
| 8 | Brain | ✅ high | remember/recall/forget/list + send/inbox (`brain/actions.go`, `brain/messaging.go`) |
| 9 | Forge | ✅ high | richer than RFC (backward gap) |
| 10 | Plans/sessions | ✅ high | plan/phase/task/session/state verbs all present |
| 11 | Fleet & sync | ✅ high | push/pull/backoff/offline-queue/pairing/poll-fallback present |
| 12 | Notifications | ❌ partial | **report-home loop out of action — HEADLINE GAP** |
| 13 | Content | ✅ high | `content.go` (931L): generate/batch/brief/schema — backward-heavy |
| 14 | Flows | ✅ high | run/flow + per-flow MCP tools + nested composition |
| 15 | Configuration | ✅ high | all RFC fields + extras (backward gap) |
| 16 | State persistence | ✅ high | queue/concurrency/registry + ghost-agent reap + in-memory fallback |
| 17 | Polyglot mapping | 🟡 | verify 1:1 Go↔PHP claims at convergence |
| 18 | Reference | n/a | doc consolidation at convergence |
