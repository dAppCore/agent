<!-- SPDX-License-Identifier: EUPL-1.2 -->
# OpenBrain — memory & messaging

**OpenBrain** gives agents persistent, workspace-scoped **memory** plus **messaging**
between agents — the durable context layer that survives a single dispatch. This page is
how to use it; the exact call sites and protections are in [callers](callers.md).

## Memory

| Tool | What it does |
|------|--------------|
| `brain_remember` | store a memory (workspace-scoped; `org`/`project` filters) |
| `brain_recall` | semantic search — embeds the query, returns best matches |
| `brain_forget` / `brain_list` | delete / list |

Recall is **semantic, not keyword**: the backend embeds the query, searches Qdrant, then
hydrates rows from MariaDB. Memories are workspace-scoped by default.

## Messaging

`agent_send` · `agent_inbox` · `agent_conversation` — how one agent hands context to
another mid-flight (complements [session handoffs](../plans/sessions.md)).

## Two transports — and the gotcha

- **Direct** (`direct.go`) — calls `/v1/brain/*`; Bearer auth, key at `~/.claude/brain.key`
  (`0600`), default-org injection, absolute-URL rejection, retry + circuit breaker.
  Results come back **inline**.
- **Bridge** (`provider.go`) — forwards to the IDE bridge over WebSocket. **Gotcha:
  `recall`/`list` return an empty body *synchronously*; results arrive async.** By design
  for the bridge path only ([known-issues](../known-issues.md)).

## In this section

- [callers](callers.md) — every Brain call site, its protections, and request/response
  shapes.
