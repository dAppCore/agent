# Codex ↔ Claude Integration Plan (Local MCP)

## Objective

Enable Codex and Claude plugins to interoperate via local MCP servers, allowing shared tools, shared ethics modal enforcement, and consistent workflows across both systems.

## Principles

- **Ethics‑first**: Axioms of Life modal is enforced regardless of entry point.
- **MCP‑first**: Prefer MCP tools over shell execution.
- **Least privilege**: Only expose required tools and limit data surface area.
- **Compatibility**: Respect Claude’s existing command patterns while enabling Codex‑native features.

## Architecture (Proposed)

1. **Codex MCP Server**
   - A local MCP server exposing Codex tools:
     - `codex_awareness`, `codex_overview`, `codex_core_cli`, `codex_safety`
     - Future: `codex_review`, `codex_verify`, `codex_qa`, `codex_ci`

2. **Claude MCP Bridge**
   - A small “bridge” config that allows Claude to call Codex MCP tools locally.
   - Claude commands can route to Codex tools for safe, policy‑compliant output.

3. **Shared Ethics Modal**
   - A single modal source file (`core-agent/codex/ethics/MODAL.md`).
   - Both Codex and Claude MCP tools reference this modal in output.

4. **Tool Allow‑List**
   - Explicit allow‑list of MCP tools shared between systems.
   - Block any tool that performs unsafe string interpolation or destructive actions.

## Implementation Steps

1. **Codex MCP Tool Expansion**
   - Add MCP tools for key workflows (review/verify/qa/ci).

2. **Claude MCP Config Update**
   - Add a local MCP server entry pointing to the Codex MCP server.
   - Wire specific Claude commands to Codex tools.

3. **Command Harmonisation**
   - Keep command names consistent between Claude and Codex to reduce friction.

4. **Testing**
   - Headless Gemini CLI tests for Codex tools.
   - Claude plugin smoke tests for bridge calls.

5. **Documentation**
   - Add a short “Interoperability” section in Codex README.
   - Document local MCP setup steps.

## Risks & Mitigations

- **Hook incompatibility**: Treat hooks as best‑effort; do not assume runtime support.
- **Policy blocks**: Avoid shell execution; use MCP tools for deterministic output.
- **Surface creep**: Keep tool lists minimal and audited.

## Success Criteria

- Claude can call Codex MCP tools locally without shell execution.
- Ethics modal is consistently applied across both systems.
- No unsafe string handling paths in shared tools.
