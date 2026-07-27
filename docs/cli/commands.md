<!-- SPDX-License-Identifier: EUPL-1.2 -->
# Command reference

The full `core-agent` command surface. For build + run modes see [the index](README.md).
Registered in `commands.go:registerApplicationCommands`.

## Chat

| Command | What it does |
|---------|--------------|
| `core-agent chat --user=<id>` | interactive Lemma REPL against a local `lthn-mlx` serve; every turn auto-captured to the user's archive ([inference](../inference/)) |

## Local engine control (the `lthn-mlx` serve)

| Command | Flags |
|---------|-------|
| `serve-status` | snapshot the serve — model, profile, context, cache, runtime |
| `serve-reload` | hot-swap the model — `--confirm=<machine-hash> --model=<path> [--profile=<name> --context=N]` |
| `serve-profiles` | list tuning profiles |
| `models-download` | queue an HF download — `--repo=<id> [--revision=<rev>] [--no-wait]` |
| `models-job` | poll a download job — `--id=<job-id>` |
| `opencode-models` | list OpenCode dispatch models (free Zen + authed Go tiers) |

These drive the engine's `/v1/admin/*` API — see [inference](../inference/).

## Containers

| Command | What it does |
|---------|--------------|
| `core-agent shell <id> [--runtime=<rt>] [--shell=<path>]` | attach a terminal to a running container/VM ([shell](../shell/)) |

## Dispatch & tracker (the `agentic:` verbs)

Every MCP dispatch/tracker tool also has a CLI verb under the `agentic:` prefix (plus a
bare alias). Examples:

| Verb | What it does |
|------|--------------|
| `agentic:issue/list` · `issue/get` · `issue/create` · `issue/comment` · `issue/assign` | work the tracker |
| `agentic:repo/sync` | freshen a repo's working tree before a dispatch |
| `agentic:plan/*` · `phase/*` · `session/*` · `sprint/*` | structured work ([plans](../plans/)) |
| `agentic:pipeline/audit` · `pipeline/epic` · `pipeline/monitor` | orchestration ([pipeline](../pipeline/)) |
| `agentic:fleet/nodes` · `fleet/status` | the fleet ([fleet](../fleet/)) |
| `agentic:workspace/stats` | permanent dispatch stats from `.core/workspace/db.duckdb` |

## Info & maintenance

| Command | What it does |
|---------|--------------|
| `version` | name + version, Go/OS/arch, home, hostname, pid, update channel |
| `check` | health — `agents.yaml` present, workspace count, services/actions/commands/env registered |
| `env` | print every `core.Env()` key + value |
| `update` | self-update on the configured channel (`update.go`) |

Global flags: `--quiet`/`-q` (errors only), `--debug`/`-d` (debug logging).
