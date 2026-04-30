---
name: capabilities
description: Return the machine-readable Codex capability manifest for ecosystem integration
---

# Capability Manifest

Use this when another tool, service, or agent needs a stable description of the Codex plugin surface.

## Preferred Sources

1. Read `core-agent/codex/.codex-plugin/capabilities.json`
2. If the Gemini extension is available, call the `codex_capabilities` tool and return its output verbatim

## What It Contains

- Plugin namespaces and command families
- Claude parity mappings for the `core` workflow
- Extension tools exposed by the Codex/Gemini bridge
- External marketplace sources used by the ecosystem
- Recommended workflow entry points for orchestration, review, QA, CI, deploy, and research

## Output

Return the manifest as JSON without commentary unless the user asks for interpretation.
