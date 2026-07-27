<!-- SPDX-License-Identifier: EUPL-1.2 -->
# Plans, phases & sessions

This is the surface for work bigger than a single dispatch: **plans** of ordered phases,
**sprints** that group them, and **sessions** that track each agent's run and hand off to
the next. Everything is exposed as MCP tools and `agentic:` CLI verbs, and persisted by
the PHP backend so work survives across machines.

## The nouns

| Noun | What it is |
|------|-----------|
| **Plan** | an ordered set of **phases** — the unit of structured work |
| **Phase** | one step within a plan |
| **Sprint** | a grouping/planning window over plans |
| **Session** | one agent's run — log, artifacts, handoff notes ([sessions](sessions.md)) |

## Plans

```
agentic:plan/create  plan/get   plan/list   plan/show   plan/status   plan/read
plan/update          plan/check plan/archive plan/delete plan/templates
```

Create from a template (`plan/templates`), drive its phases (`phase/get`, …), track with
`plan/status`, `archive` when done.

## Sprints

```
agentic:sprint/create  sprint/get  sprint/list  sprint/update  sprint/archive
```

## In this section

- [sessions](sessions.md) — the per-agent run + the handoff mechanism (the spine that
  lets agents continue each other's work).

**Related:** [dispatch](../dispatch/) (a session wraps a dispatch) · [pipeline](../pipeline/)
(orchestration produces epics/phases) · [fleet](../fleet/) (sessions resume across the
shared backend).
