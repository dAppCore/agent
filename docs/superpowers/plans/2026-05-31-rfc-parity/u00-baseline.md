<!-- SPDX-Licence-Identifier: EUPL-1.2 -->

# U0 — Baseline & gate harness

> **Sub-skill:** `superpowers:executing-plans`. Prereq for U1–U12. No production code changes —
> this unit establishes the gate, the trackers, and the loop's exit path.

**Goal:** Confirm the GOAL.md gate is runnable and green, record the baseline in `GOAL.md`, and
adopt `PARITY.md` as the living tracker, so every later unit has a known-good starting line.

**Depends on:** nothing. **Sections:** none (harness).

---

- [ ] **Step 1 — Confirm the gate is green.**

Run:
```bash
cd go && go build ./... && go vet ./... && go test ./... -count=1 -timeout 120s
```
Expected: build clean, vet clean, all packages `ok` (baseline was 14 packages green on 2026-05-31).
If anything is red, that is a *pre-existing* failure — write `BLOCKED.md` naming it and stop
(the drive assumes a green baseline).

- [ ] **Step 2 — Confirm the EXIT path is wired.**

Read `go/pkg/agentic/dispatch.go` `detectFinalStatus` and confirm a non-empty `BLOCKED.md` maps to
status `blocked` (RFC §4.6). This is the loop's free-ticket-out; it must work before relying on it.
Expected: `BLOCKED.md` present → `blocked`.

- [ ] **Step 3 — Fill `GOAL.md` Status with the baseline.**

Edit `GOAL.md`'s `## Status` section (currently an empty placeholder) to record, present-tense:
- Build/vet/test: green (14 packages).
- Forward parity: high across ~17/18 sections (see `docs/superpowers/parity/PARITY.md`).
- Open forward items: §12 (U10), §6 (U5), §13 (U11).
- Backward reconcile pending across §2/§3/§7/§9/§10/§14/§15.
Keep it present-tense, no roadmap (GOAL.md rule).

- [ ] **Step 4 — Adopt `PARITY.md` as the tracker.**

Confirm `docs/superpowers/parity/PARITY.md` exists and reflects the corrected survey. Later units
update it (residue / resolved gaps) at their Step 6.

- [ ] **Step 5 — Commit.**

```bash
git add GOAL.md docs/superpowers
git commit -m "chore(agent): baseline GOAL.md parity status + drive plan (U0)

Co-Authored-By: Virgil <virgil@lethean.io>"
```

**PASS:** gate green, `GOAL.md` Status filled, trackers in place.
