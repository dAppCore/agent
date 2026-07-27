<!-- SPDX-License-Identifier: EUPL-1.2 -->
# Local providers

Providers whose **model runs on your own machine** — against the local `lthn-mlx` engine
(or Ollama) instead of a cloud API. No data leaves the box. Detail behind
[providers](README.md).

## OpenCode against local models

`opencode:<profile>` dispatches OpenCode at a local OpenAI-compatible endpoint. The
profile names which endpoint + model — e.g. LEM profiles like `opencode:lemmy` or
`opencode:devstral`. The model server (`lthn-mlx`) must be running separately — see
[inference](../inference/). OpenCode also has **remote tiers** (the free *Zen* tier and
authed *Go* tiers) if you want them — list them with `core-agent opencode-models`.

See [opencode](../opencode/) for profile management (the `hub`'s `/profile` control plane).

## LEM / Ollama agents

The dispatch local-agent path (`localAgentCommandScript`) builds a runner against a local
model by **LEM profile** (`lemmy`, `devstral-24b`, …) or an **Ollama** model. These run
**natively on the host** and talk to the local engine directly.

## Why local

- Nothing leaves the machine — useful for private repos / air-gapped work.
- No per-token cloud cost.
- The same `lthn-mlx` engine that powers [chat](../inference/) powers dispatch.

**Related:** [inference](../inference/) (the engine + chat) · [opencode](../opencode/) ·
[remote](remote.md) (the cloud alternative).
