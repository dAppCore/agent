<!-- SPDX-License-Identifier: EUPL-1.2 -->
# Local models & chat

`core-agent` talks to a local `lthn-mlx` model engine through the `lemma` client, and
keeps every chat turn in a portable per-user archive (`chathistory`). This is the index;
[`local-inference.md`](local-inference.md) has the launch commands and
[`typologies.md`](typologies.md) has workstation sizing / safe model combinations.

## Chatting

| Surface | How |
|---------|-----|
| CLI REPL | `core-agent chat --user=<id>` — interactive chat against the local serve |
| MCP tool | `lemma_send` — a calling agent sends a message, gets a reply |

Both **auto-capture every turn** to the user's archive (below).

## The portable chat archive (continuity rights)

Every turn is written to a per-user DuckDB file:

```
~/Lethean/data/users/<id>/chats.duckdb
```

**This file is the user's property.** Changing model or provider can never take the
history away — that's the continuity-rights principle, enforced by keeping the archive
local and per-user (not in the model engine). `export.go` exports it; `migrations/`
carries the schema.

## Controlling the engine

`lemma` drives the engine's admin API (`/v1/admin/*`), surfaced as CLI commands:

| Command | Endpoint | Does |
|---------|----------|------|
| `serve-status` | `/v1/admin/serve` (+ `/machine`) | snapshot model, profile, context, cache, runtime |
| `serve-reload` | `/v1/admin/serve` | **hot-swap** the loaded model (needs `--confirm=<machine-hash>`) |
| `serve-profiles` | `/v1/admin/profiles` | list tuning profiles |
| `models-download` / `models-job` | download API | queue + poll HF model downloads |

The `--confirm=<machine-hash>` on `serve-reload` is a safety interlock so you don't
hot-swap the wrong machine's engine.

## Running OpenCode against local models

OpenCode can be dispatched against these local endpoints — see
[`../opencode/`](../opencode/) for profiles and the `opencode:<profile>` agent string.

## Next

[`local-inference.md`](local-inference.md) (launch) · [`typologies.md`](typologies.md)
(sizing) · [opencode](../opencode/) · [cli](../cli/) (the `serve-*` / `models-*` commands).
