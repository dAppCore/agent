<!-- SPDX-License-Identifier: EUPL-1.2 -->
# Review queue

When the [closeout pipeline](../pipeline/) emits `PRNeedsReview` (auto-merge is off, or a
PR needs a human/agent look), the work lands in the review queue.

| Tool | What it does |
|------|--------------|
| `agentic_review_queue` | list / work the queue of PRs awaiting review — reviewers, and the stored review output |

The queue is the human-in-the-loop seam: with `auto-merge` disabled (see
[pipeline](../pipeline/)), every PR routes here instead of merging itself. Reviewers are
assigned, and review output is stored against the PR.

## Next

[pipeline](../pipeline/) (the `PRNeedsReview` source) · [scan-mirror](../scan-mirror/)
(where findings become issues).
