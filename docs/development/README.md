---
title: Development
description: How to build, test, and contribute to core/agent — a polyglot Go + PHP repository driven by the core CLI.
---
<!-- SPDX-License-Identifier: EUPL-1.2 -->
# Development

core/agent is a **polyglot repository**: Go and PHP live side by side, each with its own
toolchain. The `core` CLI wraps both and is the primary interface for development tasks.
This section is how to build it, test it, extend it, and the standards to follow.

## Prerequisites

| Tool | Version | Purpose |
|------|---------|---------|
| Go | 1.26+ | Go packages, CLI, MCP servers |
| PHP | 8.2+ | Laravel package, Pest tests |
| Composer | 2.x | PHP dependencies |
| `core` CLI | latest | wraps both toolchains; enforced by plugin hooks |
| `jq` | any | JSON parsing in shell hooks |

Full setup (Go workspace, Composer) is in [building](building.md).

## In this section

- [building](building.md) — the Go workspace, building the binary, MCP/serve modes.
- [testing](testing.md) — Go + PHP test suites and conventions.
- [standards](standards.md) — formatting, linting, and coding standards (UK English, error patterns).
- [extending](extending.md) — adding Go packages / CLI commands / MCP tools, and PHP models / actions / controllers.
- [plugins](plugins.md) — the `provider/` plugin trees (Claude Code, Codex, …) and the hook system.
- [configuration](configuration.md) — client, PHP, and workspace config.

**Related:** [architecture](../architecture.md) (how the packages fit) ·
[providers](../providers/) (the dispatch providers these plugins back).
