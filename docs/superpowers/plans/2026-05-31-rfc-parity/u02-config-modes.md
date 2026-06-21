<!-- SPDX-Licence-Identifier: EUPL-1.2 -->

# U2 — §15 configuration + §2 binary & modes

> **Sub-skill:** `superpowers:executing-plans`. Reconcile loop (see `00-MASTER.md`).

**Goal:** §15 and §2 are present-tense-true in both directions.
**Depends on:** U1. **Sections:** §15 (RFC.md:354-372), §2 (RFC.md:34-48).

**Code to read:**
- §15: `runner/queue.go` + `agentic/queue.go` — `DispatchConfig` (`default_agent`, `runtime`,
  `image`, `gpu`, `workspace_root`), `ConcurrencyLimit`, `RateConfig` (`daily_limit`,
  `min_delay`, `sustained_delay`, `burst_window`, `burst_delay`), `AgentIdentity`.
- §2: `cmd/core-agent/commands.go` (version/check/env/chat/hub/serve-status/serve-reload/
  serve-profiles/models-download/models-job), `main.go:68` (`coremcp.Register` provides
  `mcp`/`serve`), `agentic/commands.go:31` (`run/flow` + `agentic:run/flow`).

**Known forward items:** none — all 11 modes wired, all RFC config fields parsed. Confirm.
**Known backward gaps (fold into RFC):**
- §15: add `pools`, `default_persona`, `personas`, `host_mounts` to the `agents.yaml` schema.
- §2: document the bare + `agentic:`-prefixed command-alias convention; state that `mcp`/`serve`
  are provided by the external `dappco.re/go/mcp` service (`coremcp.Register`), and that the flow
  mode is `run/flow` (slash form, flat `core.Command` API).

---

- [ ] **Step 1 — Read** the config structs and the command registrations above.
- [ ] **Step 2 — Reconcile forward.** Verify each §2 mode's behaviour matches its one-line RFC
  description; verify each §15 field is parsed and used. Fix any mismatch (TDD).
- [ ] **Step 3 — Reconcile backward.** Make the concrete RFC edits in the gaps list above; scan
  `queue.go`/`runner.go` for any further config field not in §15 and add it.
- [ ] **Step 4 — Gate:** `cd go && go build ./... && go vet ./... && go test ./... -count=1 -timeout 120s`.
- [ ] **Step 5 — Commit** `docs(agent): reconcile RFC §2/§15 to code (U2)` + Virgil trailer; include `RFC.md`.
- [ ] **Step 6 — Tracker:** tick boxes; note residue in `PARITY.md`.

**PASS:** §2/§15 zero gaps both ways; gate green.
**EXIT:** an `agents.yaml` field's intent is unclear → `BLOCKED.md`.
