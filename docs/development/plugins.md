<!-- SPDX-License-Identifier: EUPL-1.2 -->
# Provider plugins & the hook system

Per-provider integration trees live under `provider/` (the dispatch-side catalogue is
[providers](../providers/); this page is how to build them):

- `provider/claude/` — Claude Code plugin sources (`core`, `core-go`, `core-php`, `devops`,
  `infra`, `research`, plus the `camofox_mcp` and `hermes_runner_mcp` MCP plugins).
- `provider/codex/` — OpenAI Codex plugin sources (`core`, `code`, `ci`, `qa`, `review`,
  `verify`, plus `ethics`, `guardrails`, `perf`, `issue`, `coolify`, `awareness`, `api`,
  `collect`).
- `provider/google/` — Gemini CLI integration.
- `provider/hermes/` — Hermes plugins + skills (incl. the OpenBrain memory/context Python
  plugins).

## Claude Code plugins

The marketplace registry at the repo root (`.claude-plugin/marketplace.json`) publishes
the plugins. Install:

```bash
claude plugin marketplace add https://github.com/dappcore/agent
claude plugin install core
```

Each plugin lives in `provider/claude/<name>/`:

```
provider/claude/<name>/
├── .claude-plugin/plugin.json   # metadata (name, version, description)
├── 000.mcp.json                 # MCP server registration (optional)
├── hooks.json                   # hook declarations (optional)
├── scripts/                     # supporting + hook scripts (optional)
├── commands/                    # slash commands (*.md)
├── agents/                      # subagent definitions (optional)
└── skills/                      # skill definitions (optional)
```

## Hook system

The `core` plugin's `hooks.json` fires scripts (`provider/claude/core/scripts/`) across the
Claude Code lifecycle — PreToolUse guards, PostToolUse auto-format + debug warnings +
inbox/notify checks, completion checks. Hook scripts read JSON on stdin and emit a JSON
object with a `decision` (`approve` / `block`) and optional `message`. Test one by piping a
tool-input fixture into it.

## Adding a plugin

1. Create `provider/claude/<name>/.claude-plugin/plugin.json` with `name`, `description`,
   `version`, `author`, `license` (EUPL-1.2).
2. Add Markdown command files in `commands/` — the filename becomes the command name.
3. Register it in `.claude-plugin/marketplace.json` (`name`, `source`
   `./provider/claude/<name>`, `description`, `version`).
