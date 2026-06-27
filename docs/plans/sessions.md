<!-- SPDX-License-Identifier: EUPL-1.2 -->
# Sessions — the handoff spine

A **session** is one agent's run on a piece of work: a log, its artifacts, and the
**handoff notes** the next agent reads. Sessions are what let one agent pick up exactly
where another stopped. This is the detail behind [plans](README.md).

## Verbs

```
agentic:session/start     agentic:session/log       agentic:session/artifact
agentic:session/handoff   agentic:session/get       agentic:session/list
agentic:session/complete  agentic:session/end       agentic:session/continue
agentic:session/resume    agentic:session/replay
```

- `session/start` opens a session; `session/log` appends progress; `session/artifact`
  attaches outputs.
- `session/continue` / `session/resume` pick up an existing session; `session/replay`
  walks its log.

## The handoff

`session/handoff` writes the notes the next agent reads. The handoff is a structured
`Handoff` map — **but if that map is empty and plain `HandoffNotes` are set, the notes
become the handoff** (`sessionEndFromInput`). A terminal `session/end` /
`session/complete` stamps `EndedAt` and merges the handoff in.

This is one of two context-passing mechanisms; the other is [brain](../brain/) messaging
(`agent_send` / `agent_inbox`).

## Persistence

Sessions are held by the PHP backend (`/v1/sessions`), not locally — which is why a
session opened on one machine can be resumed on another across the [fleet](../fleet/).
