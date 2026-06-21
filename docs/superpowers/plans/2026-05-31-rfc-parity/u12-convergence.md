<!-- SPDX-Licence-Identifier: EUPL-1.2 -->

# U12 — Convergence pass (terminal gate)

> **Sub-skill:** `superpowers:executing-plans`. The terminal gate of the drive (GOAL.md PASS).

**Goal:** A full forward+backward pass over all of `RFC.md` finds **zero gaps in both directions**
→ the drive is done.
**Depends on:** U1–U11. **Sections:** all (§2–§18).

---

- [ ] **Step 1 — Forward sweep.** Re-read every `RFC.md` section against its code. Each described
  behaviour must be present and accurate. List any residual forward gap (should be none if U1–U11
  passed). Any found → route back to the owning unit, fix, return.
- [ ] **Step 2 — Backward sweep.** Re-scan each subsystem's code for behaviour of consequence not
  in `RFC.md`. Each found → fold a present-tense line into the right section (or flag dead code for
  removal). Should be none if U1–U11 did their Step 3.
- [ ] **Step 3 — Gate (full).**
```bash
cd go && go build ./... && go vet ./... && go test ./... -count=1 -timeout 120s
# core/lint QA gate clean
```
All must be green.
- [ ] **Step 4 — Two consecutive clean rounds.** Per GOAL.md convergence, a pass must find zero
  forward AND zero backward gaps. If this pass found any, fix and run U12 again; convergence =
  a clean pass that changed nothing.
- [ ] **Step 5 — Fill `GOAL.md` Status = PASS.** Record, present-tense: forward parity ✓, backward
  parity ✓, build/vet/test green, core/lint clean, zero gaps both ways. Remove any stale residue
  note from `PARITY.md`.
- [ ] **Step 6 — Commit.**
```bash
git add RFC.md GOAL.md docs/superpowers
git commit -m "docs(agent): RFC↔code parity convergence — GOAL.md PASS (U12)

Co-Authored-By: Virgil <virgil@lethean.io>"
```

**PASS (the whole drive):** a full pass finds zero gaps both directions; gate green; `GOAL.md`
Status reads PASS.
**EXIT:** if convergence keeps surfacing the same gap across N rounds without progress → `BLOCKED.md`
escalating it rather than grinding (GOAL.md A1).
