<!-- SPDX-Licence-Identifier: EUPL-1.2 -->

# core/agent — Implementation Goal

> **For the IDE-Opus / agentic worker:** `RFC.md` is the source of truth for what the
> code does. This file is the pass/fail gate. Drive `RFC.md` into the code, then drive the
> code's reality back into `RFC.md`, until they agree in both directions. You always have a
> clean way out — see **EXIT**. Bailing cleanly when blocked is an expected, valid outcome,
> never a failure.

## Goal

Bring the core-agent code into parity with `RFC.md` — every described behaviour present,
accurate, and tested — and keep `RFC.md` honest about what the code actually does.

## The Loop

1. **Implement** — take `RFC.md` section by section; make the code match what each says.
2. **Reconcile forward** — did this pass implement the *full* section? If the plan missed an
   adjustment, it is not done: list the gap, continue. (This is the safety-net for when a
   superpowers plan doesn't pick up every adjustment.)
3. **Reconcile backward** — once a section's code is in parity, scan that code for behaviour
   that is **not** in `RFC.md`. Real, intended behaviour → add a present-tense line to
   `RFC.md` so it is captured and **not de-prioritised**. Dead/accidental code → flag for
   removal; do not spec it.
4. **Repeat** until a full pass finds zero gaps in *both* directions (convergence).

## PASS — done (objective, machine-checkable; the gate evaluates this each round)

- Every `RFC.md` section's described behaviour is present in the code (forward parity).
- No code behaviour of consequence is absent from `RFC.md` (backward parity).
- `cd go && go build ./...` clean.
- `cd go && go test ./... -count=1` green.
- core/lint QA gate clean.
- A full pass produced **zero forward gaps AND zero backward gaps**.

## EXIT — the free ticket out (FAIL with dignity; never grind)

Write `BLOCKED.md` with a *specific* question, and stop, when:

- `RFC.md` is ambiguous or self-contradictory on something load-bearing — do not guess, ask.
- A required external (a dependency, an endpoint, a primitive) is missing or broken — report it.
- N consecutive rounds make no progress on the same gap — escalate rather than thrash.

`BLOCKED.md` → `detectFinalStatus` marks the workspace `blocked` → the loop ends and surfaces
the question. This is A1 in the loop: a defined, dignified exit always exists.

## Roles

- **Opus (in IDE)** implements + reconciles against `RFC.md`.
- **Haiku** is the cheap gate: each round, read state against this file → **continue / pass /
  exit**. Checklist-only — no judgement beyond PASS / EXIT above. When the loop runs via the
  opencode plugin, the gate reads `session.idle` (round done), `session.error` (→ EXIT), and
  build/test/lint output.

## Status

<!-- FILL after the first reconcile pass: forward gaps found, backward gaps folded into RFC.md,
     build/test/lint state, any BLOCKED.md raised. Keep present-tense; no roadmap. -->
