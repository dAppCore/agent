<!-- SPDX-Licence-Identifier: EUPL-1.2 -->

# U9 — §7 plugin providers

> **Sub-skill:** `superpowers:executing-plans`. Reconcile loop (see `00-MASTER.md`). Depends on the
> capability set the providers expose (built/confirmed in U3/U4/U6).

**Goal:** §7 is present-tense-true in both directions.
**Depends on:** U3, U4, U6. **Sections:** §7 (RFC.md:246-271). Note: `provider/` is at the **repo
root**, not under `go/`.

**Code to read:**
- `provider/claude/` — `mcp.json` (auto-registers core-agent), `hooks.json` (inbox notifications,
  auto-format), `agents/`, `commands/`, `skills/`.
- `provider/codex/` — `.codex-plugin/plugin.json` (the only `@opencode-ai/plugin`-style manifest
  the survey found), `provider/google/`, `provider/hermes/`.
- `pkg/lib/persona/` (personas that map onto agent files).

**Known forward items (verify-and-close — the real item):**
- **`provider/opencode` appears ABSENT.** A clean `ls provider/` shows `claude, codex, google,
  hermes` only; `grep -rl '@opencode-ai/plugin' provider` matched only `provider/codex`. But RFC §7
  (and CLAUDE.md) describe `provider/opencode` as a core deliverable (the `@opencode-ai/plugin`
  with `tool()` exports + `session.*` hooks). **Step 1 must verify this first.** Three outcomes:
  (a) it exists somewhere the survey missed → reconcile; (b) it was relocated (git log:
  "relocate opencode + provider backend — Mantis #1807") → point the RFC at the new home;
  (c) it is genuinely missing → forward gap: build it per §7, **or** correct §7 to match reality.
**Known backward gaps (fold into RFC §7):** the RFC frames "two providers" (claude + opencode) but
`provider/` carries **codex, google, hermes** too. Reconcile: describe the full set, or clarify
that codex/google/hermes are distinct from the *plugin* providers. Confirm the two-layer dispatch
(opencode `Task` subagents + core-agent cross-host fleet) and the `POST /mcp` hub-attach are described.

---

- [ ] **Step 1 — Locate the providers.** `ls provider/` and
  `grep -rl '@opencode-ai/plugin' provider .` to settle the `provider/opencode` question (present /
  relocated / missing). Read `provider/claude` + whatever the opencode plugin resolves to +
  `pkg/lib/persona`.
- [ ] **Step 2 — Reconcile forward / close.** Verify the Claude plugin (MCP/hooks/agents/commands/
  skills) matches §7. Then close the opencode-plugin item per its resolved outcome (a/b/c above):
  reconcile, re-point the RFC, or build/correct. Verify personas≡agent-defs and skills≡SKILL.md.
- [ ] **Step 3 — Reconcile backward.** Resolve the provider-set framing (codex/google/hermes) in
  §7; fold any plugin capability not described.
- [ ] **Step 4 — Gate:** `cd go && go build ./... && go vet ./... && go test ./... -count=1 -timeout 120s`
  (plus any provider-side lint/test, e.g. `provider/opencode` package scripts).
- [ ] **Step 5 — Commit** `docs(agent): reconcile RFC §7 plugin providers to code (U9)` + Virgil trailer.
- [ ] **Step 6 — Tracker:** tick boxes; note residue in `PARITY.md`.

**PASS:** §7 zero gaps both ways; gate green.
**EXIT:** the codex/google/hermes providers' role contradicts the RFC's two-provider model
load-bearingly → `BLOCKED.md`.
