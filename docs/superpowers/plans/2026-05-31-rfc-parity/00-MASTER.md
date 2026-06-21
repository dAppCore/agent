<!-- SPDX-Licence-Identifier: EUPL-1.2 -->

# core/agent RFC↔code Parity Drive — Master Plan

> **For agentic workers:** REQUIRED SUB-SKILL — use `superpowers:subagent-driven-development`
> (recommended) or `superpowers:executing-plans` to run this plan unit-by-unit. Each per-unit file
> uses checkbox (`- [ ]`) steps. This is the **drive-target loop** described in `GOAL.md`.

**Goal:** Bring `core-agent` into full RFC↔code parity in both directions — every behaviour in
`RFC.md` present/accurate/tested, and no code behaviour of consequence missing from `RFC.md` —
until a full pass finds zero gaps either way.

**Architecture:** A survey + verify-first pass (`docs/superpowers/parity/PARITY.md`) established
that the code is **already at high forward parity** (build/vet/test green; ~17 of 18 sections
present). So this drive is **reconcile-dominated**: mostly *backward* reconcile (fold real code
behaviour into `RFC.md`), one real forward-build (§12 report-home), and two verify-and-close items
(§6 proxy coverage, §13 content). Work is decomposed into 13 dependency-ordered units; each runs
the GOAL.md loop over its section(s) to its own PASS.

**Tech Stack:** Go (module `dappco.re/go/agent`, root `go/`), the `core` framework
(`core.Command`/`core.Action`/`core.Result`, `coreio`, `coreerr`), DuckDB/go-store, MCP, opencode,
PHP platform (out of scope here except where the RFC names a Go↔PHP bridge).

---

## How to execute one unit (the GOAL.md loop)

Every unit (except U0/U12) is the same procedure applied to its section(s). The per-unit file
pre-loads the **concrete** gaps the survey already found so the steps are real, not placeholders.

- [ ] **Step 1 — Read the contract.** Read the unit's `RFC.md` section(s) and the listed code.
- [ ] **Step 2 — Reconcile forward.** For each behaviour the RFC describes, confirm the code does
  it. If a described behaviour is missing/partial → implement it (TDD: failing test → minimal
  code → green). The per-unit file lists the known forward items.
- [ ] **Step 3 — Reconcile backward.** Scan the unit's code for behaviour of consequence **not**
  in `RFC.md`. Real/intended → add a present-tense line to the relevant `RFC.md` section. Dead/
  accidental → flag for removal (do not spec it). The per-unit file lists the known backward gaps.
- [ ] **Step 4 — Run the gate** (see below). Must be green.
- [ ] **Step 5 — Commit** with a conventional message + the Virgil trailer.
- [ ] **Step 6 — Update trackers.** Tick the unit's boxes; note residue (if any) in `PARITY.md`.
- [ ] **EXIT (always available):** if the RFC is ambiguous/self-contradictory on something
  load-bearing, or a required external is missing, or N rounds make no progress — write
  `BLOCKED.md` with a *specific* question and stop. Bailing cleanly is a valid outcome, not a
  failure (GOAL.md A1).

## The gate (GOAL.md PASS criteria)

```bash
cd go && go build ./...                       # clean
cd go && go vet ./...                          # clean
cd go && go test ./... -count=1 -timeout 60s   # green
# core/lint QA gate clean — as run by §5 step 1 (agentic.qa = core/lint + build + test)
```

A unit PASSes when: its sections have forward parity, its backward gaps are folded into `RFC.md`,
and the gate is green. The **drive** PASSes (U12) when a full pass finds **zero gaps both ways**.

## Conventions

- **UK English** (colour, organisation, initialise). **SPDX** `// SPDX-License-Identifier: EUPL-1.2`
  on every new file. **Errors:** `coreerr.E("pkg.Method", "msg", err)` (3 args), never `fmt.Errorf`.
  **File I/O:** `coreio.Local` / `WriteMode(path, content, 0600)`, never `os.ReadFile/WriteFile`.
- **Commits:** `type(scope): description` + `Co-Authored-By: Virgil <virgil@lethean.io>`.
- **RFC edits are first-class deliverables** — backward reconcile means *editing `RFC.md`*, and that
  is the point of the drive, not a side effect.

---

## Unit index (dependency order)

| Unit | Sections | Kind | Depends on | File |
|------|----------|------|-----------|------|
| U0 | — | baseline & gate | — | `u00-baseline.md` |
| U1 | §3, §16 | reconcile (foundation) | U0 | `u01-domain-state.md` |
| U2 | §15, §2 | reconcile | U1 | `u02-config-modes.md` |
| U3 | §4 | reconcile | U1, U2 | `u03-dispatch.md` |
| U4 | §5 | reconcile | U3 | `u04-completion.md` |
| U5 | §6 | verify-and-close | U1, U2 | `u05-opencode.md` |
| U6 | §8, §9 | reconcile (backward-heavy) | U1 | `u06-brain-forge.md` |
| U7 | §10, §14 | reconcile (backward-heavy) | U1, U6 | `u07-plans-flows.md` |
| U8 | §11 | reconcile | U1, U6 | `u08-fleet-sync.md` |
| U9 | §7 | reconcile | U3, U4, U6 | `u09-providers.md` |
| U10 | §12 | **implement (headline)** | U9 | `u10-report-home.md` |
| U11 | §13, §17, §18 | verify + close-out | U6 | `u11-content-polyglot-docs.md` |
| U12 | all | convergence gate | U1–U11 | `u12-convergence.md` |

## Known forward items (the only code-build work)

1. **§12 report-home loop** (U10) — restore the push-listener → plugin-surface path. HEADLINE.
2. **§6.5 proxy coverage / `prompt_async`** (U5) — verify the proxy forwards the full session API;
   close any uncovered path; decide on a typed async client.
3. **§13 content surface** (U11) — confirm `content.go` exposes `content.generate`/`content.batch`,
   else correct the RFC.

## Known backward-gap registry (concrete fold-into-RFC tasks)

These are the survey's confirmed "code does more than the RFC says" items. Each is a concrete edit
to `RFC.md`, executed in the owning unit:

- **U2/§15:** add `pools`, `default_persona`, `personas`, `host_mounts` to the `agents.yaml` schema.
- **U2/§2:** document the bare + `agentic:`-prefixed command-alias convention; note `mcp`/`serve`
  come from the external `coremcp.Register` service.
- **U6/§9:** add `issue/assign`, `issue/report`, `repo/get`, `repo/list`, `repo/sync`.
- **U7/§10:** add `plan/from-issue`, `plan/templates`, `plan/check`, status aliases.
- **U7/§14:** add per-flow MCP-tool auto-registration (Mantis #1806) + nested flow composition with
  cycle+depth guards (Mantis #1805).
- **U9/§7:** reconcile the two-provider framing with the actual `provider/` set
  (claude, codex, google, hermes, opencode).
- **U1/§3:** confirm/annotate the Go↔PHP split (all listed models exist in Go).
- (further backward gaps are expected per unit — Step 3 surfaces them.)

## Self-review

- **Spec coverage:** every `RFC.md` section maps to a unit (U1–U11 cover §2–§18; U0 baseline, U12
  convergence). ✓
- **No placeholders:** forward items and backward gaps are named concretely with file/section refs;
  reconcile steps are a real procedure, not "TBD". ✓
- **Consistency:** unit numbering, dependencies, and the `PARITY.md` gap map agree. ✓
