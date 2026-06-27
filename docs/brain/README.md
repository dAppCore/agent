<!-- SPDX-License-Identifier: EUPL-1.2 -->
# OpenBrain — durable memory & cross-agent messaging

`brain` is the client for OpenBrain: persistent, workspace-scoped memory plus messaging
between agents. This guide is how to use it; the exact call sites, protections, and
request/response shapes are in [`callers.md`](callers.md).

## Memory tools

| Tool | What it does |
|------|--------------|
| `brain_remember` | store a memory (workspace-scoped; `org`/`project` filters) |
| `brain_recall` | semantic search — embeds the query, returns the best matches |
| `brain_forget` | delete a memory |
| `brain_list` | list memories |

Recall is semantic, not keyword: the backend embeds your query, searches Qdrant, then
hydrates the rows from MariaDB. Memories are **workspace-scoped** — one workspace can't
see another's unless you widen the `org`/`project` filter.

## Messaging tools

| Tool | What it does |
|------|--------------|
| `agent_send` | send a message to another agent |
| `agent_inbox` | read your inbox |
| `agent_conversation` | a threaded conversation between agents |

This is how one agent hands context to another mid-flight (complements session handoffs —
see [plans](../plans/)).

## Two transports — and the one gotcha

The same tools run over either transport:

- **Direct** (`direct.go`) — calls `/v1/brain/{remember,recall,forget,list}` on the API.
  Hardened: Bearer auth, **default-org injection**, the key at `~/.claude/brain.key`
  (`0600`), **absolute-URL rejection**, retry with jitter, and a **circuit breaker**.
  Results come back **inline**.
- **Bridge** (`provider.go`) — forwards to the IDE bridge over WebSocket
  (`NewProvider(bridge, hub)`). **Gotcha: in bridge mode, `recall`/`list` return an
  empty body *synchronously* — the real results arrive asynchronously over the
  WebSocket.** This is by design for the bridge path and only affects bridge-mode
  clients; the `DirectSubsystem` path returns results inline. (See
  [`../known-issues.md`](../known-issues.md).)

## Backend (for context)

The PHP `BrainService` is the canonical write/read path: it writes to MariaDB first and
queues async indexing (`EmbedMemory`) into **Qdrant + Elasticsearch**; recall embeds the
query, searches Qdrant, hydrates from MariaDB. Qdrant is authenticated with an `api-key`
header.

## Next

[`callers.md`](callers.md) (every call site + its protections) · [plans](../plans/)
(session handoffs, the other context-passing mechanism).
