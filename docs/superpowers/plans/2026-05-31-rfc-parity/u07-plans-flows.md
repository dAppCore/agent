<!-- SPDX-Licence-Identifier: EUPL-1.2 -->

# U7 — §10 plans/sessions + §14 flows (backward-heavy)

> **Sub-skill:** `superpowers:executing-plans`. Reconcile loop (see `00-MASTER.md`). Backward-heavy:
> extra plan verbs + flow features the RFC doesn't yet describe.

**Goal:** §10 and §14 are present-tense-true in both directions.
**Depends on:** U1, U6. **Sections:** §10 (RFC.md:296-306), §14 (RFC.md:346-352).

**Code to read:**
- §10: `commands_plan.go` (plan/create, plan/from-issue, plan/templates, plan/list, plan/get,
  plan/read, plan/show, plan/update, plan/status, plan/update_status, plan/check),
  `commands_phase.go` (phase/get, phase/update_status, phase/add_checkpoint + aliases),
  `commands_task.go` (task/create, task/update, task/toggle), session + state commands,
  `template.go` (PlanTemplateVersion render).
- §14: `flow.go`, `flow_tools.go` (per-flow MCP-tool auto-registration, Mantis #1806),
  `pkg/lib/flow/` (path-addressed YAML), `agentic/commands.go` (`run/flow`), nested composition
  with cycle+depth guards (Mantis #1805).

**Known forward items:** none — plan/phase/task/session/state verbs + flow run/compose present.
Confirm session.{start,continue,end,handoff,replay} and state.{set,get,list,delete} match §10.
**Known backward gaps (fold into RFC):**
- §10: add `plan/from-issue`, `plan/templates`, `plan/check`, the `plan/status`↔`plan/update_status`
  aliases.
- §14: add per-flow MCP-tool auto-registration (1806) and nested flow composition with cycle+depth
  guards (1805); note the declared Inputs schema with run-time validation (Mantis #1804).

---

- [ ] **Step 1 — Read** the §10 command files + §14 flow files.
- [ ] **Step 2 — Reconcile forward.** Verify each §10 lifecycle verb + §14 flow capability
  (sequential/parallel/conditional `when:`/agent-dispatch/manual-approval, `--dry-run`, `--var`).
  Fix mismatches (TDD).
- [ ] **Step 3 — Reconcile backward.** Make the §10/§14 edits above; add further behaviour found.
- [ ] **Step 4 — Gate:** `cd go && go build ./... && go vet ./... && go test ./... -count=1 -timeout 120s`.
- [ ] **Step 5 — Commit** `docs(agent): reconcile RFC §10/§14 plans+flows to code (U7)` + Virgil trailer.
- [ ] **Step 6 — Tracker:** tick boxes; note residue in `PARITY.md`.

**PASS:** §10/§14 zero gaps both ways; gate green.
**EXIT:** a flow primitive's semantics are RFC-vs-code contradictory → `BLOCKED.md`.
