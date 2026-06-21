<!-- SPDX-Licence-Identifier: EUPL-1.2 -->

# U5 — §6 opencode surface (verify-and-close)

> **Sub-skills:** `superpowers:systematic-debugging` (to confirm proxy coverage) + `test-driven-
> development` (to close any gap). Reconcile loop (see `00-MASTER.md`). This unit has a real
> verify-and-close item, not just reconcile.

**Goal:** §6 (6.1–6.6) is present-tense-true, with the §6.5 session-API surface actually reachable.
**Depends on:** U1 (Sandbox), U2 (config). **Sections:** §6 (RFC.md:181-244).

**Code to read:**
- `opencode/generate.go` (§6.1 Generate — sync `/session` + `/session/:id/message`),
  `agentic/opencode.go` + `agentic/provider_manager.go` (§6.1 ProviderManager in-process backend).
- `opencode/opencode.go` (§6.2 lifecycle Start/Stop, SSE eventEmitter), `opencode/reconcile.go`
  (§6.2 Reconcile — adopt only this install's labelled containers).
- `opencode/profile.go` (§6.3 profile→endpoint map + `CORE_OPENCODE_*` overrides + wire config).
- `opencode/proxy.go` (§6.5/§6.6 proxy path set), `opencode/control.go` (§6.6 ControlGroup),
  `cmd/core-agent/commands_hub.go` (§6.6 hub edge — already high parity).

**Known forward items (verify-and-close):**
1. **Proxy coverage** — `proxy.go` declares `/session`, `/global/event`, `/config`. Verify (prefix
   match) it forwards the full §6.5 surface: `/session/:id/prompt_async`, `/children`, `/abort`,
   `/fork`, `/permissions`, **`POST /mcp`**, `/agent`, `/command`, `/global/health`. Any path not
   covered → add it (TDD: a `proxy_reject_test.go`-style test that asserts the path forwards).
2. **`prompt_async`** — core-agent's `Generate` is sync. Decide: is a typed no-wait client needed
   for the fleet, or is proxy-passthrough sufficient? Implement or correct the RFC §6.5 wording.
**Known backward gaps (fold into RFC §6):** extra control-group routes (spawn/list/stop/inspect/
upgrade/enable/studio/tui) in `control.go`; the audit-edge wiring already in `commands_hub.go`.

---

- [ ] **Step 1 — Read** the §6 files; map the proxy path set vs the §6.5 list.
- [ ] **Step 2 — Reconcile forward / close.** Close item 1 (proxy coverage) and decide item 2
  (`prompt_async`) with a test. Verify lifecycle/profiles/permission-boundary match 6.2–6.4.
- [ ] **Step 3 — Reconcile backward.** Fold the extra control routes + audit edge into §6.
- [ ] **Step 4 — Gate:** `cd go && go build ./... && go vet ./... && go test ./... -count=1 -timeout 120s`.
- [ ] **Step 5 — Commit** `feat/docs(agent): close §6 opencode proxy coverage + reconcile (U5)` + Virgil trailer.
- [ ] **Step 6 — Tracker:** tick boxes; note residue in `PARITY.md`.

**PASS:** §6 zero gaps both ways; the §6.5 surface is reachable through the proxy; gate green.
**EXIT:** `prompt_async` requires an upstream opencode-serve capability that isn't present →
`BLOCKED.md` naming it.
