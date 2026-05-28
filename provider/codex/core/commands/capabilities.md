---
name: capabilities
description: Return the machine-readable Codex capability manifest for ecosystem integration
---

# Capability Manifest

Use this when another tool, service, or agent needs a stable description of the Codex plugin surface.

## Preferred Sources

1. Read `provider/codex/.codex-plugin/capabilities.json`
2. If the Gemini extension is available, call the `codex_capabilities` tool and return its output verbatim
3. If the manifest is unavailable, summarise the command files in `provider/codex/core/commands/`

## What It Contains

- Plugin namespaces and command families
- CoreAgent command families exposed to Codex
- MCP tool and CLI fallback preferences
- External marketplace sources used by the ecosystem
- Recommended workflow entry points for orchestration, plans, sessions, review, QA, platform sync, content, deploy, and research

## Output

Return the manifest as JSON without commentary unless the user asks for interpretation.
