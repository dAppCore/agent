<!-- SPDX-License-Identifier: EUPL-1.2 -->
# Orchestration pipeline

The higher-level **audit → epic → monitor** flow that turns raw issues into structured,
dispatched work — the layer that decides *what* to dispatch. This is the detail behind
[pipeline](README.md).

Exposed as MCP tools and `agentic:pipeline/*` CLI verbs:

| Verb | Stage |
|------|-------|
| `pipeline/audit` | **Stage 1** — audit issues into implementation work (extract + link findings) |
| `pipeline/epic` | **Stages 2–3** — epic orchestration (group work into epics, fan out) |
| `pipeline/monitor` | watch open PRs and **auto-intervene** (e.g. resolve stuck PRs) |

The pipeline is staged so a run can stop and resume: `audit` produces findings, `epic`
groups them into dispatchable work, `monitor` keeps the in-flight PRs moving.

This produces the epics/phases that [plans](../plans/) track; [dispatch](../dispatch/)
does the running; the [closeout](closeout.md) pipeline does the finishing.
