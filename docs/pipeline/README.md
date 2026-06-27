<!-- SPDX-License-Identifier: EUPL-1.2 -->
# Pipeline

core/agent has **two pipelines**, and keeping them apart is the key to understanding the
system:

1. **[Closeout](closeout.md)** — what runs *per dispatch* once an agent finishes:
   QA → PR → verify → merge, message-driven and `auto-*` gated.
2. **[Orchestration](orchestration.md)** — the higher-level *audit → epic → monitor* flow
   that turns raw issues into dispatched work.

The orchestration pipeline decides **what** to dispatch; [dispatch](../dispatch/) does the
**running**; the closeout pipeline does the **finishing**. Findings from closeout can feed
back as new issues for orchestration to pick up — a closed loop.

## In this section

- [closeout](closeout.md) — the per-dispatch QA→PR→verify→merge stages, the `auto-*`
  gates, and the "no checks ⇒ no auto-merge" safety.
- [orchestration](orchestration.md) — `pipeline/audit`, `pipeline/epic`,
  `pipeline/monitor`.

**Related:** [dispatch](../dispatch/) · [review](../review/) (the `PRNeedsReview` path) ·
[scan-mirror](../scan-mirror/) · [plans](../plans/).
