<!-- SPDX-Licence-Identifier: EUPL-1.2 -->

# U4 — §5 completion pipeline

> **Sub-skill:** `superpowers:executing-plans`. Reconcile loop (see `00-MASTER.md`).

**Goal:** §5 is present-tense-true in both directions.
**Depends on:** U3. **Sections:** §5 (RFC.md:160-179).

**Code to read:**
- `agentic/actions.go:199` (`agent.completion` Task composition), `:347` (`agentic.ingest`).
- `agentic/qa.go` (step 1: core/lint + build + test, capture every finding to workspace DuckDB).
- `agentic/auto_pr.go` (step 2: open PR).
- the verify handler (step 3: CI + review → `PRMerged`/`PRNeedsReview` — grep `cmdVerify`/`PRMerged`).
- `agentic/commands.go:79` (`poke` — step 5 drain queue).
- `agentic/commit.go` (step 6: workspace DuckDB → go-store journal).
- `poindexter.go` (`clusterFindings` across tool/severity/file/category/frequency; diff vs prior;
  new/resolved/persistent) + `report.go` (`.meta/report.json`).

**Known forward items:** none expected — 6-step chain + Poindexter + report.json present. Confirm
the "QA captures raw findings, no filtering during" principle and the journal-then-purge ordering.
**Known backward gaps (fold into RFC §5):** surface during Step 3 (e.g. push-failure recording in
`auto_pr.go:52/63/82`, extra async steps).

---

- [ ] **Step 1 — Read** the completion chain + Poindexter + report.
- [ ] **Step 2 — Reconcile forward.** Verify the 6 steps fire in order with the right async-ness;
  verify Poindexter clusters in N-dimensional space and diffs against prior cycles; verify raw
  DuckDB is journalled then purged. Fix mismatches (TDD).
- [ ] **Step 3 — Reconcile backward.** Add RFC lines for consequential behaviour not in §5.
- [ ] **Step 4 — Gate:** `cd go && go build ./... && go vet ./... && go test ./... -count=1 -timeout 120s`.
- [ ] **Step 5 — Commit** `docs(agent): reconcile RFC §5 completion pipeline to code (U4)` + Virgil trailer.
- [ ] **Step 6 — Tracker:** tick boxes; note residue in `PARITY.md`.

**PASS:** §5 zero gaps both ways; gate green.
**EXIT:** the verify→merge criteria are RFC-vs-code contradictory → `BLOCKED.md`.
