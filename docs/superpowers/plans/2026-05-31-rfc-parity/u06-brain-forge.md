<!-- SPDX-Licence-Identifier: EUPL-1.2 -->

# U6 — §8 brain + §9 forge (backward-heavy)

> **Sub-skill:** `superpowers:executing-plans`. Reconcile loop (see `00-MASTER.md`). Backward-heavy:
> §9's command surface is richer than the RFC documents.

**Goal:** §8 and §9 are present-tense-true in both directions.
**Depends on:** U1. **Sections:** §8 (RFC.md:273-285), §9 (RFC.md:287-295).

**Code to read:**
- §8: `brain/actions.go` (handleRemember/Recall/Forget/List/Send/Inbox), `brain/direct.go`,
  `brain/messaging.go`, `brain/tools.go` (BrainMemory=Memory). Note CLAUDE.md gotcha: recall/list
  are async bridge proxies — empty responses are intentional, not a bug.
- §9: `agentic/commands_forge.go` (issue/{get,list,comment,create,assign,report,update,archive},
  pr/{get,list,merge,close}, repo/{get,list,sync}, branch/delete), the scan + mirror handlers.

**Known forward items:** none — all §8 verbs + §9 forge ops present. Confirm the brain bridge
(Go) ↔ PHP store split matches §8 (don't audit PHP/Qdrant depth).
**Known backward gaps (fold into RFC §9):** add `issue/assign`, `issue/report`, `repo/get`,
`repo/list`, `repo/sync` (RFC §9 currently lists only get/list/create/update/comment/archive + pr
+ branch.delete + scan + mirror). Note the bare + `agentic:`-prefixed alias convention.

---

- [ ] **Step 1 — Read** the §8 brain files and §9 `commands_forge.go`.
- [ ] **Step 2 — Reconcile forward.** Verify §8 remember→embed→upsert / recall→embed→search→
  hydrate semantics are described correctly (Go bridge only); verify §9's listed ops exist. Fix
  mismatches (TDD).
- [ ] **Step 3 — Reconcile backward.** Make the §9 edits above; add any further forge/brain
  behaviour not in the RFC.
- [ ] **Step 4 — Gate:** `cd go && go build ./... && go vet ./... && go test ./... -count=1 -timeout 120s`.
- [ ] **Step 5 — Commit** `docs(agent): reconcile RFC §8/§9 brain+forge to code (U6)` + Virgil trailer.
- [ ] **Step 6 — Tracker:** tick boxes; note residue in `PARITY.md`.

**PASS:** §8/§9 zero gaps both ways; gate green.
**EXIT:** the brain bridge's Go↔PHP contract is ambiguous in the RFC → `BLOCKED.md`.
