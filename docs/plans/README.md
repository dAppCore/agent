<!-- SPDX-License-Identifier: EUPL-1.2 -->
# Plans, Phases & Sessions — structured multi-agent work

This is the surface for work that's bigger than one dispatch: ordered phases, grouped
sprints, and per-agent sessions that hand off to the next agent. Everything is exposed
both as MCP tools and as `agentic:` CLI verbs, and persisted via the PHP backend
(`/v1/plans`, `/v1/sessions`, `/v1/sprints`).

## The nouns

| Noun | What it is |
|------|-----------|
| **Plan** | an ordered set of **phases** — the unit of structured work |
| **Phase** | one step within a plan |
| **Sprint** | a grouping of work (a planning window) |
| **Session** | one agent's run: a **log**, **artifacts**, and **handoff notes** for whoever picks it up next |

## Plans

```
agentic:plan/create   agentic:plan/get     agentic:plan/list     agentic:plan/show
agentic:plan/status   agentic:plan/read    agentic:plan/update   agentic:plan/check
agentic:plan/archive  agentic:plan/delete  agentic:plan/templates
```

Create from a template (`plan/templates` lists them), drive its phases (`phase/get`, …),
track progress with `plan/status`, `archive` when done.

## Sessions — the handoff spine

A session tracks an agent's work so another agent can continue it:

```
agentic:session/start     agentic:session/log       agentic:session/artifact
agentic:session/handoff   agentic:session/get       agentic:session/list
agentic:session/complete  agentic:session/end       agentic:session/continue
agentic:session/resume    agentic:session/replay
```

- `session/start` opens a session; `session/log` appends progress; `session/artifact`
  attaches outputs.
- **`session/handoff` writes the handoff** — the notes the next agent reads.
  **Nuance:** the handoff is a structured `Handoff` map, but if it's empty and plain
  `HandoffNotes` are set, **the notes become the handoff** (`sessionEndFromInput`).
  A terminal `session/end`/`session/complete` stamps `EndedAt` and merges the handoff.
- `session/continue` / `session/resume` pick up where one stopped; `session/replay`
  walks the log.

## Sprints

```
agentic:sprint/create  agentic:sprint/get  agentic:sprint/list
agentic:sprint/update  agentic:sprint/archive
```

Group plans/work into a sprint window for planning and reporting.

## Persistence

State is held by the PHP backend, not locally — `/v1/plans`, `/v1/plans/{slug}/phases`,
`/v1/sessions`, `/v1/sprints`. That's why a session opened on one machine can be resumed
on another (the fleet shares the backend).

## Next

[dispatch](../dispatch/) (sessions wrap a dispatch) · [pipeline](../pipeline/) (the
orchestration pipeline produces epics/phases) · [fleet](../fleet/) (cross-machine, shared
backend).
