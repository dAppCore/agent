<!-- SPDX-License-Identifier: EUPL-1.2 -->
# Providers

A **provider** is the coding agent you dispatch work to — named in the `provider[:model]`
[agent string](../dispatch/runners.md). core/agent integrates several, and the useful
split is **where the model runs**: a **remote** provider calls a cloud API; a **local**
provider runs against your own `lthn-mlx` engine.

There's a second, independent axis — **where the *process* runs** (native on the host vs
in a container) — covered in [dispatch/runners](../dispatch/runners.md).

## The matrix

| Provider | Model | Process | What it is |
|----------|-------|---------|-----------|
| `claude` | [remote](remote.md) — Anthropic | host | Claude Code |
| `codex` | [remote](remote.md) — OpenAI | container | OpenAI Codex |
| `gemini` | [remote](remote.md) — Google | container | Gemini CLI |
| `vibe` | [remote](remote.md) — Mistral | host | Mistral Vibe CLI bridge |
| `coderabbit` | [remote](remote.md) | host | review |
| `opencode` | [local](local.md) (or remote tiers) | host | OpenCode against `lthn-mlx` |
| `hermes` | provider integration | — | Python plugins + skills |

Each provider integration lives under `provider/<name>/` in the repo.

## In this section

- [remote](remote.md) — the cloud providers (claude, codex, gemini, vibe, coderabbit).
- [local](local.md) — running agents against your own models (opencode + LEM/ollama).

**Related:** [dispatch/runners](../dispatch/runners.md) (native vs container) ·
[inference](../inference/) (the local engine) · [opencode](../opencode/).
