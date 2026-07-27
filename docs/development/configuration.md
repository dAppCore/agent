<!-- SPDX-License-Identifier: EUPL-1.2 -->
# Project configuration

## Go client config (`~/.core/agentic.yaml`)

```yaml
base_url: https://api.lthn.sh
token: your-api-token
default_project: my-project
agent_id: cladius
```

Environment variables `AGENTIC_BASE_URL`, `AGENTIC_TOKEN`, `AGENTIC_PROJECT`, and
`AGENTIC_AGENT_ID` override the YAML values.

## PHP config

The service provider merges two config files on boot:

- `src/php/config.php` into the `mcp` config key (brain database, Ollama URL, Qdrant URL)
- `src/php/agentic.php` into the `agentic` config key (Forgejo URL, token, general settings)

Environment variables:

| Variable | Purpose |
|----------|---------|
| `ANTHROPIC_API_KEY` | Claude API key |
| `GOOGLE_AI_API_KEY` | Gemini API key |
| `OPENAI_API_KEY` | OpenAI API key |
| `BRAIN_DB_HOST` | dedicated brain database host |
| `BRAIN_DB_DATABASE` | dedicated brain database name |

## Workspace config (`.core/workspace.yaml`)

Controls `core` CLI behaviour when running from the repository root:

```yaml
version: 1
active: core-php
packages_dir: ./packages
settings:
  suggest_core_commands: true
  show_active_in_prompt: true
```
