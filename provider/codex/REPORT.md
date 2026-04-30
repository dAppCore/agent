# Codex Plugin Parity Report

## Summary

Feature parity with the Claude plugin suite has been implemented for the Codex plugin set under `core-agent/codex`.

## What Was Implemented

### Marketplace & Base Plugin

- Added Codex marketplace registry at `core-agent/codex/.codex-plugin/marketplace.json`.
- Updated base Codex plugin metadata to `0.1.1`.
- Embedded the Axioms of Life ethics modal and “no silly things with strings” guardrails in `core-agent/codex/AGENTS.md`.

### Ethics & Guardrails

- Added ethics kernel files under `core-agent/codex/ethics/kernel/`:
  - `axioms.json`
  - `terms.json`
  - `claude.json`
  - `claude-native.json`
- Added `core-agent/codex/ethics/MODAL.md` with the operational ethics modal.
- Added guardrails guidance in `core-agent/codex/guardrails/AGENTS.md`.

### Plugin Parity (Claude → Codex)

For each Claude plugin, a Codex counterpart now exists with commands, scripts, and hooks mirrored from the Claude example (excluding `.claude-plugin` metadata):

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

Each Codex sub-plugin includes:
- `AGENTS.md` pointing to the ethics modal and guardrails
- `.codex-plugin/plugin.json` manifest
- Mirrored `commands/`, `scripts/`, and `hooks.json` where present

### Gemini Extension Alignment

- Codex ethics modal and guardrails embedded in Gemini MCP tools.
- Codex awareness tools return the modal content without shell execution.

## Known Runtime Constraints

- Gemini CLI currently logs unsupported hook event names (`PreToolUse`, `PostToolUse`). Hooks are mirrored for parity, but hook execution depends on runtime support.
- Shell-based command prompts are blocked by Gemini policy; MCP tools are used instead for Codex awareness.
- `claude/code/hooks.json` is not valid JSON in the upstream source; the Codex mirror preserves the same structure for strict parity. Recommend a follow-up fix if you want strict validation.

## Files & Locations

- Codex base: `core-agent/codex/`
- Codex marketplace: `core-agent/codex/.codex-plugin/marketplace.json`
- Ethics modal: `core-agent/codex/ethics/MODAL.md`
- Guardrails: `core-agent/codex/guardrails/AGENTS.md`

## Next Artefacts

- `core-agent/codex/IMPROVEMENTS.md` — improvements beyond Claude capabilities
- `core-agent/codex/INTEGRATION_PLAN.md` — plan to integrate Codex and Claude via local MCP
