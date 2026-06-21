<!-- SPDX-Licence-Identifier: EUPL-1.2 -->

# Design — core/agent RFC↔code Parity Drive

**Date:** 2026-05-31 · **Author:** Cladius (Opus)
**Decisions:** scope = full parity drive (decomposed) · sequencing = **dependency order** ·
deliverable = **master + per-unit plan files**

## Context

`RFC.md` is the present-tense contract for the `core-agent` Go binary; `GOAL.md` is the RFC↔code
parity gate (forward + backward parity, `BLOCKED.md` free-ticket-out, Haiku round-gate). A survey
+ verify-first pass (recorded in `docs/superpowers/parity/PARITY.md`) found:

- **Build / vet / test: GREEN** (14 packages `ok`, 0 vet findings).
- **Forward parity is HIGH** across ~17 of 18 sections — the RFC reads as written *from* the code.
  Verify-first corrected several first-pass over-calls (§3, §7, §10, §11, §13 are present).
- This is therefore a **reconcile-dominated drive**: the bulk of the work is *backward* reconcile
  (fold real code behaviour the RFC omits into `RFC.md`), with **one clear forward-code item**
  (§12 report-home loop) and **two verify-and-close items** (§6 `prompt_async`/proxy coverage;
  §13 content surface).

## Goal

Bring the code into parity with `RFC.md` in both directions until a full pass finds zero gaps
either way. **PASS** = the GOAL.md gate: forward parity, backward parity, `go build ./...` clean,
`go test ./... -count=1` green, core/lint clean, zero gaps both directions.

## Approach

- **Engine:** the GOAL.md loop per unit — implement → reconcile forward → reconcile backward →
  PASS, with `BLOCKED.md` as the dignified exit when a unit hits ambiguity or a missing external.
- **Sequencing:** **dependency order** — foundations → consumers → the §12 headline → close-out.
- **Decomposition:** 13 units (U0–U12). Each unit is independently executable, scoped to a
  section or section-group, with its own PASS (build/test/lint green + zero gaps for its sections).
- **Per-unit shape:** because most units are reconcile, each plan file is the GOAL.md loop applied
  to its section(s), **pre-loaded with the concrete backward gaps the survey already found** (so
  the tasks are real, not placeholders). U10 (§12) carries real implementation tasks.
- **Deliverable:** a master plan + one detailed plan file per unit under `docs/superpowers/plans/`.

## Units (dependency order)

### U0 — Baseline & gate harness *(prereq)*
Fill `GOAL.md` Status from the survey/verify findings; confirm the gate commands run and the
`BLOCKED.md` → `detectFinalStatus` → `blocked` path + Haiku round-gate are wired; adopt
`PARITY.md` as the living tracker.

### U1 — §3 domain model + §16 state persistence *(foundation)*
Reconcile types + persistence. Backward: confirm/annotate the Go↔PHP split. Confirm
queue/concurrency/registry groups + ghost-agent reap + in-memory fallback against §16.

### U2 — §15 configuration + §2 binary & modes
Backward: fold `pools`, `default_persona`, `personas`, `host_mounts` into §15; document the bare +
`agentic:`-prefixed command-alias convention; clarify `mcp`/`serve` external-service provenance in §2.

### U3 — §4 dispatch & workspace
Reconcile 4.1–4.6 (prep, prompt build, agent commands, container exec, queue/concurrency/rate,
outcome/bail). Fold backward gaps.

### U4 — §5 completion pipeline
Reconcile the 6-step chain + Poindexter clustering + DuckDB lifecycle. Fold backward gaps.

### U5 — §6 opencode surface *(verify-and-close)*
Verify the proxy covers the full §6.5 surface (`prompt_async`, `/children`, `/abort`, `/fork`,
`/permissions`, `POST /mcp`, `/agent`, `/command`, `/global/health`); close any uncovered path.
Decide whether the fleet needs a typed async client; implement or correct the RFC. Reconcile
lifecycle/profiles/permission-boundary.

### U6 — §8 brain + §9 forge
Backward: fold §9's extra verbs (`issue/assign`, `issue/report`, `repo/{get,list,sync}`) into the
RFC. Confirm brain bridge async semantics. Reconcile.

### U7 — §10 plans/sessions + §14 flows
Backward: fold `plan/from-issue`, `plan/templates`, `plan/check`, per-flow MCP tools, nested flow
composition into the RFC. Reconcile.

### U8 — §11 fleet & sync
Reconcile push/pull/backoff/offline-queue/pairing/poll-fallback against §11. Fold backward gaps.

### U9 — §7 plugin providers
Reconcile the `provider/claude` + `provider/opencode` surfaces against the Go capability set from
U3/U4/U6. Note: `provider/` also carries codex/google/hermes — reconcile the RFC's two-provider
framing with the actual provider set.

### U10 — §12 report-home loop *(headline implementation)*
Investigate the exact break in the push-listener → plugin-surface loop (emit side exists in
`message.go`/`monitor.go`; consumer side in the plugins) and restore it so inbox +
dispatched-agent progress reach the orchestrator again. TDD where the seam allows.

### U11 — §13 content + §17 polyglot + §18 reference
Verify `content.go` exposes `content.generate`/`content.batch` (or correct the RFC); verify the
§17 1:1 Go↔PHP map; consolidate the §18 doc tree.

### U12 — Convergence pass
A full forward+backward scan finds zero gaps in both directions → GOAL.md PASS; fill `GOAL.md`
Status with the convergence result.

## Dependencies (build-order rationale)

U0 precedes all. U1 (types/state) underpins everything. U2 (config/modes) underpins dispatch.
U3→U4 is the doing-path then its completion. U5/U6 are consumers of types+config. U7/U8 are
orchestration + fleet. U9 (plugins) depends on the capability set (U3/U4/U6). U10 (report-home)
depends on U9 (plugins are the surface). U11 is cross-cutting close-out. U12 is the terminal gate.

## Acceptance

- **Per unit:** the unit's sections satisfy forward + backward parity; `go build`/`go test`/core-lint
  green; backward gaps folded into `RFC.md`.
- **Overall:** U12 finds zero gaps both ways; `GOAL.md` Status reflects PASS.

## References

- `RFC.md` — the contract (drive-target)
- `GOAL.md` — the parity gate + loop + EXIT
- `docs/superpowers/parity/PARITY.md` — the corrected survey/gap map this design is built on
