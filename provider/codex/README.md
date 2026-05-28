# Host UK Codex Plugin

This plugin provides Codex-friendly context and guardrails for the **core-agent** monorepo. It mirrors key behaviours from the Claude plugin suite, focusing on safe workflows, the Host UK toolchain, and the Axioms of Life ethics modal.

## Plugins

- `awareness`
- `ethics`
- `guardrails`
- `api`
- `ci`
- `code`
- `collect`
- `coolify`
- `core`
- `issue`
- `perf`
- `qa`
- `review`
- `verify`

## What It Covers

- CoreAgent orchestration commands for workspaces, plans, sessions, Forge, platform sync, content, and QA
- Core CLI enforcement (Go/PHP via `core`)
- UK English conventions
- Safe shell usage guidance
- Pointers to shared scripts from `core-agent/claude/code/`

## Usage

Include `core-agent/codex` in your workspace so Codex can read `AGENTS.md` and apply the guidance.

## Files

- `AGENTS.md` - primary instructions for Codex
- `scripts/awareness.sh` - quick reference output
- `scripts/overview.sh` - README output
- `scripts/core-cli.sh` - core CLI mapping
- `scripts/safety.sh` - safety guardrails
- `.codex-plugin/plugin.json` - plugin metadata
- `.codex-plugin/marketplace.json` - Codex marketplace registry
- `.codex-plugin/capabilities.json` - machine-readable command and integration manifest
- `ethics/MODAL.md` - ethics modal (Axioms of Life)
