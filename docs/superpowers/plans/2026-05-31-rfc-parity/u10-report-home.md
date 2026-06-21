<!-- SPDX-Licence-Identifier: EUPL-1.2 -->

# U10 — §12 report-home loop (headline)

> **Sub-skills:** `superpowers:systematic-debugging` (the loop is broken — find the cause before
> fixing), then `superpowers:test-driven-development` for the fix. This is the one unit with real
> forward-build work; the exact fix depends on investigation, so this plan is investigate→debug→
> TDD, not pre-written code (writing pre-written code for an undiagnosed break would be a guess).

**Goal:** Restore the report-home loop so new inbox messages and dispatched-agent progress reach
the orchestrator again through the Claude / opencode plugins (RFC §12).

**Depends on:** U9 (plugin providers are the surface the loop reports to).

**Sections:** §12 (RFC.md:323-334). RFC §12 self-acknowledges: *"this loop is currently out of
action and needs restoring."*

**Known-present pieces (emit side):**
- `go/pkg/messages/messages.go:95` — `InboxMessage` struct.
- `go/pkg/agentic/message.go:98` — emits `messages.InboxMessage` via `Core().ACTION(...)`;
  `message.go:166` references `coremcp.ChannelInboxMessage`.
- `go/pkg/monitor/monitor.go:493` — emits `InboxMessage` (dispatched-agent progress).

**Known-present pieces (consumer side):**
- `provider/claude/hooks.json` — inbox-notification hook.
- `provider/opencode/src/*` — `session.*` event hooks (`session.idle`→done, `session.error`→
  BLOCKED, `tool.execute.after`→progress) feeding the report-home loop.

---

- [ ] **Step 1 — Map the loop end to end.**

Read, in order: `messages.go` (`InboxMessage` + `ChannelInboxMessage`), `agentic/message.go`
(`cmdMessageSend`/`cmdMessageInbox`/`cmdMessageConversation` + the `ACTION` emit), `monitor.go:480-510`
(progress emit), the host-side push listener (search the MCP host for the `InboxMessage` / push
consumer — `grep -rn 'InboxMessage\|PushNotification\|ChannelInboxMessage' go/ provider/`), and the
plugin consumers (`provider/claude/hooks.json`, `provider/opencode/src`).
Write the actual wiring as a short diagram in your working notes: *emit → channel/IPC → listener →
plugin hook → orchestrator surface.*

- [ ] **Step 2 — Locate the break.**

Identify which hop is dead. Candidate failure points (confirm which, do not assume):
  - the `ACTION(InboxMessage{...})` is emitted but nothing subscribes to `ChannelInboxMessage`;
  - the push listener exists but isn't started in the relevant mode (`mcp`/`hub`);
  - the plugin hook (`hooks.json` / opencode `session.*`) no longer points at a live handler;
  - a channel/struct field renamed on one side only.
Record the exact file:line of the break.

- [ ] **Step 3 — Write a failing test that reproduces the break.**

Add a test at the seam you found (e.g. `go/pkg/agentic/message_test.go` or the listener's package):
emit an `InboxMessage` and assert the listener/surface receives it. It must FAIL now, demonstrating
the break.
Run: `cd go && go test ./pkg/<pkg>/ -run TestReportHome -v` → Expected: FAIL.

- [ ] **Step 4 — Fix minimally (TDD).**

Reconnect the dead hop with the smallest change that makes the test pass. Follow existing patterns
(`Core().ACTION`, `coremcp.Channel*`, the plugin hook contract). No `fmt.Errorf` — use
`coreerr.E`. Re-run the test → Expected: PASS.

- [ ] **Step 5 — Verify the full loop.**

Exercise emit → surface across the real boundary the RFC describes (orchestrator sees inbox +
dispatched-agent progress through the plugin). If a plugin (`provider/claude` or
`provider/opencode`) needs a hook reconnected, do it here and note it in U9's scope.
Run the gate: `cd go && go build ./... && go vet ./... && go test ./... -count=1 -timeout 120s`.

- [ ] **Step 6 — Reconcile RFC §12.**

The loop is live again → **remove the "currently out of action / needs restoring" note** from
`RFC.md` §12 and make the description present-tense-true. Fold any newly-surfaced behaviour
(backward gap) into §12.

- [ ] **Step 7 — Commit.**

```bash
git add go/pkg docs/superpowers RFC.md provider
git commit -m "fix(agent): restore the report-home loop — push listener to plugin surface (U10, RFC §12)

Co-Authored-By: Virgil <virgil@lethean.io>"
```

**PASS:** report-home loop verified end-to-end; RFC §12 no longer flags it broken; gate green.
**EXIT:** if the break is in an external (the plugin host's IPC contract, a missing MCP channel
primitive) you cannot fix from this repo → write `BLOCKED.md` naming the exact missing piece.
