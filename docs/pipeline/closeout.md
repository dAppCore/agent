<!-- SPDX-License-Identifier: EUPL-1.2 -->
# Closeout pipeline

What runs **per dispatch** once a runner finishes: a typed IPC pipeline
(`pkg/messages/`) drives QA → PR → verify → merge. This is the detail behind
[pipeline](README.md).

## The message flow

```
AgentStarted → AgentCompleted → QAResult → PRCreated → PRMerged
                                         ↘ PRNeedsReview        ↘ WorkspacePushed
```

The messages *are* the contract. Others on the bus: `QueueDrained`, `PokeQueue`,
`SpawnQueued`, `RateLimitDetected`, `HarvestComplete` / `HarvestRejected`, `InboxMessage`.

## Stages and their `auto-*` gates

Each stage is gated by an `auto-*` config flag, so an operator can disable any of them:

| Stage | Gate | When off |
|-------|------|----------|
| QA | `auto-qa` | findings reported, no PR auto-created |
| Create PR | `auto-create` | pushed branch left for a human to PR |
| Verify | `auto-verify` | PR created but not auto-checked |
| Merge | `auto-merge` | PR left open for human merge |
| Ingest findings | `auto-ingest` | QA findings not pushed back as issues |

**Safety nuance:** a PR whose checks aren't "successful" — **including a PR with no
reported checks at all — must not auto-merge.** "No checks" is treated as not-successful
on purpose, so an unverified change never merges itself.

With `auto-ingest` on, QA findings become tracker issues that [scan](../scan-mirror/) then
picks up — closing the loop. With `auto-merge` off, PRs route to the
[review queue](../review/).
