<!-- SPDX-Licence-Identifier: EUPL-1.2 -->

# U3 — §4 dispatch & workspace

> **Sub-skill:** `superpowers:executing-plans` (+ `test-driven-development` for any forward fix).
> Reconcile loop (see `00-MASTER.md`). §4 is the largest section (4.1–4.6).

**Goal:** §4 is present-tense-true in both directions.
**Depends on:** U1, U2. **Sections:** §4 (RFC.md:75-158).

**Code to read:**
- `agentic/prep.go` (§4.1 workspace prep: `PrepInput`/`PrepOutput`, local-mirror clone, ff-only
  re-prep, `agent/{slug}` branch, specs/ + docs copy).
- `agentic/prompt.go` (§4.2 `buildPrompt` ordering).
- `agentic/agent_command.go` (§4.3 the 6 agent command shapes: claude/codex/gemini/coderabbit/
  opencode/local).
- `agentic/container.go` (§4.4 `containerCommandFor`: docker/podman/apple flags, mounts, creds,
  env, `--add-host`, gpu, `sh -c` guard + `chmod`, runtime auto-detect apple→docker→podman).
- `agentic/queue.go` + `runner/queue.go` (§4.5 queue drain, concurrency per pool + per model,
  rate daily/min/sustained/burst).
- `agentic/dispatch.go` (§4.6 `detectFinalStatus`: BLOCKED.md→blocked, nonzero→failed, else
  completed; failure backoff 3<60s→30min).

**Known forward items:** none expected — all 4.1–4.6 machinery present. Confirm depth, esp. the
command-shape flag tables (§4.3) and container flag shape (§4.4) match the RFC exactly.
**Known backward gaps (fold into RFC §4):** surface during Step 3 — e.g. extra `PrepInput` fields,
extra runtimes, extra prompt sections, `repo/sync` mirror-freshening interplay.

---

- [ ] **Step 1 — Read** the §4 files above subsection by subsection.
- [ ] **Step 2 — Reconcile forward.** For each of 4.1–4.6, diff the RFC's described behaviour
  against the code. Where the code's command/flag/ordering differs from the RFC table, decide:
  fix code (if RFC is right) or fold into RFC (if code is right). Use TDD for code fixes.
- [ ] **Step 3 — Reconcile backward.** Add present-tense RFC lines for any consequential behaviour
  not in §4 (extra fields, extra runtime handling, extra prompt context).
- [ ] **Step 4 — Gate:** `cd go && go build ./... && go vet ./... && go test ./... -count=1 -timeout 120s`.
- [ ] **Step 5 — Commit** `docs(agent): reconcile RFC §4 dispatch/workspace to code (U3)` + Virgil trailer.
- [ ] **Step 6 — Tracker:** tick boxes; note residue in `PARITY.md`.

**PASS:** §4 (all of 4.1–4.6) zero gaps both ways; gate green.
**EXIT:** a command/flag shape is RFC-vs-code contradictory and load-bearing → `BLOCKED.md`.
