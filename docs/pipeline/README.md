<!-- SPDX-License-Identifier: EUPL-1.2 -->
# Pipeline — closeout + orchestration

There are two "pipelines" in core/agent, and it helps to keep them apart:

1. **The closeout pipeline** — what runs *per dispatch* once an agent finishes
   (QA → PR → verify → merge).
2. **The orchestration pipeline** — the higher-level *audit → epic → monitor* flow that
   turns raw issues into dispatched work.

## 1. The closeout pipeline (per dispatch)

When a dispatched runner finishes, completion is detected and a **typed IPC pipeline**
(`pkg/messages/`) drives the stages. The messages *are* the contract:

```
AgentStarted → AgentCompleted → QAResult → PRCreated → PRMerged
                                         ↘ PRNeedsReview        ↘ WorkspacePushed
```

Other messages on the bus: `QueueDrained`, `PokeQueue`, `SpawnQueued`,
`RateLimitDetected`, `HarvestComplete` / `HarvestRejected`, `InboxMessage`.

### Stages and their `auto-*` gates

The flow is **AgentCompleted → QA → auto-PR → verify → merge**, and **each stage is
gated by an `auto-*` config flag**, so an operator can disable any stage independently:

| Stage | Gate | Effect when off |
|-------|------|-----------------|
| QA | `auto-qa` | findings are reported but no PR is auto-created |
| Create PR | `auto-create` | the pushed branch is left for a human to PR |
| Verify | `auto-verify` | PR is created but not auto-checked |
| Merge | `auto-merge` | PR is left open for human merge |
| Ingest findings | `auto-ingest` | QA findings are not pushed back to the tracker as issues |

**Safety nuance:** a PR whose checks are not "successful" — including **a PR with no
reported checks at all — must not auto-merge**. "No checks" is treated as not-successful
on purpose, so an unverified change never merges itself.

Findings from QA can be **ingested back into the tracker as issues** (`auto-ingest`),
closing the loop: an agent's review of one issue can spawn the next.

## 2. The orchestration pipeline (audit → epic → monitor)

A separate, higher-level surface (MCP tools + `agentic:pipeline/*` CLI verbs) turns
issues into structured, dispatched work:

| Verb | Stage |
|------|-------|
| `pipeline/audit` (`agentic:pipeline/audit`) | **Stage 1** — audit issues into implementation work (extract findings, link them) |
| `pipeline/epic` (`agentic:pipeline/epic`) | **Stages 2–3** — epic orchestration (group work into epics, fan out) |
| `pipeline/monitor` (`agentic:pipeline/monitor`) | watch open PRs and **auto-intervene** (e.g. resolve stuck PRs) |

This is the layer that decides *what* to dispatch; [dispatch](../dispatch/) does the
*running*; the closeout pipeline above does the *finishing*.

## Next

[dispatch](../dispatch/) (what triggers closeout) · [review](../review/) (the
`PRNeedsReview` path) · [scan-mirror](../scan-mirror/) (where ingested findings land) ·
[plans](../plans/) (epics/phases the orchestration produces).
