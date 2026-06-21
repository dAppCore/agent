<!-- SPDX-Licence-Identifier: EUPL-1.2 -->

# U1 — §3 domain model + §16 state persistence (foundation)

> **Sub-skill:** `superpowers:executing-plans`. Reconcile loop (see `00-MASTER.md` → "How to
> execute one unit"). Foundation — types + persistence underpin every later unit.

**Goal:** §3 and §16 are present-tense-true in both directions.
**Depends on:** U0. **Sections:** §3 (RFC.md:50-73), §16 (RFC.md:374-386).

**Code to read:**
- §3 types: `plan.go` (AgentPlan=Plan), `phase.go` (AgentPhase=Phase), `session.go`
  (AgentSession=Session), `message.go` (AgentMessage), `auth.go` (AgentApiKey), `issue.go`,
  `sprint.go`, `prompt_version.go` (Prompt/PromptVersion), `template.go` (PlanTemplateVersion),
  `state.go` (WorkspaceState), `brain/tools.go` (BrainMemory=Memory), `opencode/types.go` (Sandbox).
- §16: `statestore.go` (in-memory fallback `:40`/`:111`), `runtime_state.go`, `persist.go`,
  `queue.go` (queue/concurrency/registry groups), `prep.go:454` (ghost-agent reap).

**Known forward items:** none expected — all types + state groups present. Confirm parity.
**Known backward gaps (fold into RFC §3):** confirm/annotate which models are Go vs PHP-backed
(survey found all listed types exist in Go); confirm the supersession-chain + soft-delete fields
match `BrainMemory` reality.

---

- [ ] **Step 1 — Read** the §3 type files and §16 store files above; confirm each RFC behaviour.
- [ ] **Step 2 — Reconcile forward.** For each §3 model: verify fields/statuses match the RFC
  (e.g. AgentPlan statuses `draft/active/in_progress/needs_verification/verified/completed/archived`;
  `Sandbox` id/image/hostPort/status/created_at persisted via ORM). For §16: verify the three
  groups (queue/concurrency/registry) survive restart, dead-PID reap → `failed`, and the in-memory
  fallback path. Any mismatch → fix (TDD).
- [ ] **Step 3 — Reconcile backward.** Add present-tense RFC lines for any Go field/behaviour not
  in §3/§16 (annotate Go↔PHP split; note `prep.go` ghost-agent reap wording matches §16).
- [ ] **Step 4 — Gate:** `cd go && go build ./... && go vet ./... && go test ./... -count=1 -timeout 120s`.
- [ ] **Step 5 — Commit** `docs(agent): reconcile RFC §3/§16 to code (U1)` + Virgil trailer; include `RFC.md`.
- [ ] **Step 6 — Tracker:** tick boxes; note residue in `PARITY.md`.

**PASS:** §3/§16 zero gaps both ways; gate green.
**EXIT:** RFC ambiguous on a model's source-of-truth (Go vs PHP) you can't resolve → `BLOCKED.md`.
