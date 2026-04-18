# core/agent/

Agent dispatch, pipeline, runner service, plugins, topology.

## Specs

| File | Purpose |
|------|---------|
| [RFC.md](RFC.md) | Agent system (dispatch, daemon, tray, team model) |
| [RFC.pipeline.md](RFC.pipeline.md) | **Pipeline commands** — audit→epic→execute, MetaReader, knowledge accumulation |
| [RFC.topology.md](RFC.topology.md) | Agent topology (Cladius, Charon, local/remote) |
| [RFC.agents-brand.md](../../lthn/RFC.agents-brand.md) | Agent brand identities (in lthn/) |
| [RFC.plugin-restructure.md](RFC.plugin-restructure.md) | Plugin restructure plan |

## Subdirectories

### [flow/](flow/)
Flow system — YAML-defined agent workflows, path-addressed, composable.

### [plugins/](plugins/)
Plugin architecture — Claude, Codex, Gemini, PHP (63 commands/skills).

## Cross-References

| Spec | Relationship |
|------|-------------|
| `code/core/go/agent/RFC.md` | Go implementation (dispatch, workspace, MCP) |
| `code/core/php/agent/RFC.md` | PHP implementation (OpenBrain, content pipeline, sessions) |
| `code/core/mcp/RFC.md` | MCP transport layer agent uses |
| `code/core/config/RFC.md` | `.core/agent.yaml` config spec |
| `project/lthn/ai/RFC.md` | lthn.sh platform (fleet dispatch target) |
| `project/lthn/lem/RFC.md` | LEM training pipeline (agent findings → training data) |
