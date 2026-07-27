<!-- SPDX-License-Identifier: EUPL-1.2 -->
# Remote providers

Providers whose **model runs in the cloud** — you dispatch to them and they call out to a
hosted API. Detail behind [providers](README.md).

| Provider | Vendor | Process | Notes |
|----------|--------|---------|-------|
| `claude` | Anthropic | **host** (native) | Claude Code — plugin sets under `provider/claude/` (core, core-go, core-php) |
| `codex` | OpenAI | **container** | OpenAI Codex (`provider/codex/`) |
| `gemini` | Google | **container** | Gemini CLI (`provider/google/`) |
| `vibe` | Mistral | host | Mistral Vibe CLI bridged to the hub — exposes all core-agent MCP tools, with report-home lifecycle hooks (`provider/vibe/`) |
| `coderabbit` | — | host | review provider |

## Where they run

`claude`, `vibe`, and `coderabbit` run **natively on the host**; `codex` and `gemini` run
**inside a container** (Docker / Apple-VZ / Podman). Containerised providers reach the
host — including a local model server — via `host.docker.internal` (the dispatch adds
`--add-host=host.docker.internal:host-gateway`). See
[dispatch/runners](../dispatch/runners.md).

## Auth

Cloud providers authenticate with their vendor (API keys / CLI login) on the machine that
runs them — credentials are **not** entered through core/agent. A dispatch just selects
the provider; the provider's own CLI handles auth.

**Related:** [local](local.md) (the local-model alternative) · [dispatch](../dispatch/).
