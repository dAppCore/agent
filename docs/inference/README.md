<!-- SPDX-License-Identifier: EUPL-1.2 -->
# Local models & chat

core/agent runs against a **local `lthn-mlx` model engine** through the `lemma` client,
and keeps every chat turn in a portable per-user archive. This is the overview; the launch
commands and sizing live in the detail pages below.

## Chatting

| Surface | How |
|---------|-----|
| CLI | `core-agent chat --user=<id>` — interactive REPL against the local serve |
| MCP | `lemma_send` — an agent sends a message, gets a reply |

Both **auto-capture every turn** to `~/Lethean/data/users/<id>/chats.duckdb`.

## Continuity rights

That DuckDB archive **is the user's property** — changing model or provider can never take
the history away, because it's kept local and per-user (not in the engine). `export.go`
exports it.

## Engine control

`lemma` drives the engine's `/v1/admin/*` API via the `serve-status` / `serve-reload`
(hot-swap, with a `--confirm=<machine-hash>` interlock) / `serve-profiles` /
`models-download` commands — see [commands](../cli/commands.md).

## In this section

- [local-inference](local-inference.md) — launch commands + runner notes.
- [typologies](typologies.md) — workstation sizing + safe model combinations.
- [opencode](../opencode/) — dispatching OpenCode against these local endpoints.
