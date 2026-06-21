<!-- SPDX-Licence-Identifier: EUPL-1.2 -->

# U8 — §11 fleet & remote sync

> **Sub-skill:** `superpowers:executing-plans`. Reconcile loop (see `00-MASTER.md`).

**Goal:** §11 is present-tense-true in both directions.
**Depends on:** U1, U6. **Sections:** §11 (RFC.md:308-321).

**Code to read:**
- `fleet_connect.go` (connect + SSE/poll fallback — `:169` "fleet poll fallback exited"),
  `fleet_mode.go`, `fleet_login.go` + `auth.go` (pairing-code exchange / `AgentApiKey` bootstrap),
  `sync.go` (`/v1/agent/sync` push `:356`, `/v1/agent/context` pull `:175`, `syncBackoffSchedule`
  `:70`), `remote_sync_queue.go` (offline queue), `platform.go` + `platform_tools.go` +
  `commands_platform.go` (fleet task next/result, capabilities, heartbeat).

**Known forward items:** none expected — connect/pair/SSE+poll/sync-push-pull/offline-backoff
present. Confirm: capability registration, heartbeat, `GET /v1/fleet/task/next` polling fallback,
backoff 1s→5min (`sync.go` caps at 30s for the legacy path — reconcile the two backoff schedules
against §11's "1s → 5min" wording), and "no API key = fully offline; sync additive."
**Known backward gaps (fold into RFC §11):** the two distinct backoff schedules
(`syncBackoffSchedule` vs `remoteSyncQueueBackoff`); any platform tool not in §11.

---

- [ ] **Step 1 — Read** the fleet + sync files above.
- [ ] **Step 2 — Reconcile forward.** Verify pairing→register→SSE-jobs(+poll fallback)→heartbeat→
  report, and sync push/pull + offline queue with backoff. Reconcile the backoff numbers (RFC says
  1s→5min; code caps a path at 30s) — fix code or correct RFC. TDD for code fixes.
- [ ] **Step 3 — Reconcile backward.** Fold the extra backoff schedule + platform tools into §11.
- [ ] **Step 4 — Gate:** `cd go && go build ./... && go vet ./... && go test ./... -count=1 -timeout 120s`.
- [ ] **Step 5 — Commit** `docs(agent): reconcile RFC §11 fleet+sync to code (U8)` + Virgil trailer.
- [ ] **Step 6 — Tracker:** tick boxes; note residue in `PARITY.md`.

**PASS:** §11 zero gaps both ways; gate green.
**EXIT:** the fleet API contract (endpoints/SSE shape) can't be verified without the live
`api.lthn.ai` and the RFC is ambiguous → `BLOCKED.md`.
