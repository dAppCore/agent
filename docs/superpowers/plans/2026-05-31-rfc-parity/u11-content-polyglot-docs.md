<!-- SPDX-Licence-Identifier: EUPL-1.2 -->

# U11 — §13 content + §17 polyglot + §18 reference (verify + close-out)

> **Sub-skill:** `superpowers:executing-plans` (+ TDD if §13 needs a Go surface). Reconcile loop
> (see `00-MASTER.md`).

**Goal:** §13, §17, §18 are present-tense-true in both directions.
**Depends on:** U6. **Sections:** §13 (RFC.md:336-344), §17 (RFC.md:388-397), §18 (RFC.md:399-415).

**Code to read:**
- §13: `agentic/content.go` (the file exists; survey didn't confirm `content.generate`/
  `content.batch` verbs). Confirm what it exposes.
- §17: cross-cutting — the claimed 1:1 map (`pkg/brain/*` ↔ `Actions/Brain/*`,
  `agentic/dispatch.go` ↔ `DispatchCommand`, `agentic/actions.go` ↔ `Mcp/Tools/*`).
- §18: `docs/` tree (the sub-specs §18 references).

**Known forward items (verify-and-close):**
- §13: confirm `content.go` exposes `content.generate` + `content.batch` (and `content.schema.
  generate`). If present → reconcile. If absent → either add the thin Go surface (TDD) or correct
  §13 to "PHP-only, no Go action."
**Known backward gaps:** surface during Step 3.

---

- [ ] **Step 1 — Read** `content.go`; verify the §17 mapping spot-checks; list the §18 doc tree.
- [ ] **Step 2 — Reconcile forward.** Close the §13 content-surface question (add or correct).
  Verify the §17 1:1 claims hold (each named Go path ↔ PHP counterpart exists or is noted).
- [ ] **Step 3 — Reconcile backward.** Fold content behaviour not in §13; correct any stale §17/§18
  pointer; ensure §18's references resolve.
- [ ] **Step 4 — Gate:** `cd go && go build ./... && go vet ./... && go test ./... -count=1 -timeout 120s`.
- [ ] **Step 5 — Commit** `docs(agent): reconcile RFC §13/§17/§18 to code (U11)` + Virgil trailer.
- [ ] **Step 6 — Tracker:** tick boxes; note residue in `PARITY.md`.

**PASS:** §13/§17/§18 zero gaps both ways; gate green.
**EXIT:** §13's Go-vs-PHP intent is ambiguous and the PHP side can't be confirmed → `BLOCKED.md`.
