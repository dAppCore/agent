# Codex Extension Improvements (Beyond Claude Capabilities)

## Goal

Identify enhancements for the Codex plugin suite that go beyond Claude’s current capabilities, while preserving the Axioms of Life ethics modal and the blue-team posture.

## Proposed Improvements

1. **MCP-First Commands**
   - Replace any shell-bound prompts with MCP tools for safe, policy‑compliant execution.
   - Provide structured outputs for machine‑readable pipelines (JSON summaries, status blocks).

2. **Ethics Modal Enforcement**
   - Add a lint check that fails if prompts/tools omit ethics modal references.
   - Provide a `codex_ethics_check` MCP tool to verify the modal is embedded in outputs.

3. **Strings Safety Scanner**
   - Add a guardrail script or MCP tool to flag unsafe string interpolation patterns in diffs.
   - Provide a “safe string” checklist to be auto‑inserted in risky tasks.

4. **Cross‑Repo Context Index**
   - Build a lightweight index of core-agent plugin commands, scripts, and hooks.
   - Expose a MCP tool `codex_index_search` to query plugin capabilities.

5. **Deterministic QA Runner**
   - Provide MCP tools that wrap `core` CLI for Go/PHP QA with standardised output.
   - Emit structured results suitable for CI dashboards.

6. **Policy‑Aware Execution Modes**
   - Add command variants that default to “dry‑run” and require explicit confirmation.
   - Provide a `codex_confirm` mechanism for high‑impact changes.

7. **Unified Release Metadata**
   - Auto‑generate a Codex release manifest containing versions, commands, and hashes.
   - Add a “diff since last release” report.

8. **Learning Loop (Non‑Sensitive)**
   - Add a mechanism to collect non‑sensitive failure patterns (e.g. hook errors) for improvement.
   - Ensure all telemetry is opt‑in and redacts secrets.

## Constraints

- Must remain EUPL‑1.2.
- Must preserve ethics modal and blue‑team posture.
- Avoid shell execution where possible in Gemini CLI.
