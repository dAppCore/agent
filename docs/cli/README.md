<!-- SPDX-License-Identifier: EUPL-1.2 -->
# CLI & Binary — getting started

`core-agent` is a single Go binary that is **both an MCP server** (so IDEs and other
agents drive it) **and a CLI**. This guide is how to build it and what every command
does. Subsystem detail lives in the sibling guides linked at the bottom.

## Build & install

```bash
cd go
go build ./cmd/core-agent/      # produces ./core-agent
go install ./cmd/core-agent/    # installs to $GOPATH/bin
```

Cross-compile for the homelab Linux box (Charon):

```bash
cd go
GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -o core-agent-linux ./cmd/core-agent/
```

**The binary is dual-named.** It reads its own name from `argv[0]`
(`main.go:detectBinaryName`): invoked as `core-agent` it is `core-agent` (the legacy
default); symlinked or installed as `lthn-agent` it identifies as `lthn-agent` — the
`lthn-{mlx,cuda,amd,agent}` family naming. Same behaviour either way; only the banner,
version output, and admin-token prefixes change.

## Server modes

| Command | Transport | Use it for |
|---------|-----------|-----------|
| `core-agent mcp` | MCP over **stdio** | IDE integration — what Claude Code etc. connect to. From the shared `dappco.re/go/mcp` service. |
| `core-agent serve` | MCP over **HTTP** | cross-agent communication, CI, the homelab fleet. Also from the shared service. |
| `core-agent hub` | loopback HTTP + MCP HTTP/SSE | the agent **hub** — a loopback control plane (opencode + brain) plus the MCP tool plane. Registered by the agent itself (`commands.go`). |

`mcp` and `serve` come from `coremcp.Register` (the shared MCP service the binary wires
in `main.go`); the rest of the commands below are registered directly by
`cmd/core-agent` in `commands.go:registerApplicationCommands`.

## Talking to a model

| Command | What it does |
|---------|--------------|
| `core-agent chat --user=<id>` | Interactive Lemma REPL against a local `lthn-mlx` serve; every turn is auto-captured to the user's portable archive. See [`../inference/`](../inference/). |

## Local engine control (the `lthn-mlx` serve)

| Command | Flags |
|---------|-------|
| `core-agent serve-status` | snapshot the serve config — model, profile, context, cache, runtime |
| `core-agent serve-reload` | hot-swap the loaded model — `--confirm=<machine-hash> --model=<path> [--profile=<name> --context=N]` |
| `core-agent serve-profiles` | list tuning profiles the engine sees |
| `core-agent models-download` | queue an HF download — `--repo=<id> [--revision=<rev>] [--no-wait]` |
| `core-agent models-job` | poll a download job — `--id=<job-id>` |
| `core-agent opencode-models` | list OpenCode dispatch models (free Zen + authed Go tiers) |

These drive the engine's `/v1/admin/*` API — see [`../inference/`](../inference/).

## Containers, dispatch & structured work

- `core-agent shell <id> [--runtime=<rt>] [--shell=<path>]` — drop into a running
  container/VM. See [`../shell/`](../shell/).
- **The dispatch + tracker surface is also exposed as CLI verbs** under the `agentic:`
  prefix — e.g. `agentic:issue/list`, `agentic:issue/create`, `agentic:repo/sync`,
  `agentic:workspace/stats`, `agentic:commit`. Every MCP dispatch/tracker tool has a
  matching `agentic:<tool>` CLI verb (and a bare `<tool>` alias). See
  [`../dispatch/`](../dispatch/) and [`../plans/`](../plans/).

## Info & maintenance

| Command | What it does |
|---------|--------------|
| `core-agent version` | name + version, Go/OS/arch, home, hostname, pid, update channel |
| `core-agent check` | health check — `agents.yaml` present, workspace root + count, services/actions/commands/env-keys registered |
| `core-agent env` | print every `core.Env()` key and value |
| `core-agent update` | self-update on the configured channel (`update.go`) |

Global flags: `--quiet`/`-q` (errors only), `--debug`/`-d` (debug logging) — handled in
`commands.go:applyLogLevel` before dispatch.

## Config & layout

- **`agents.yaml`** — fleet + agent config (`agentic.AgentsConfigPath()`). `check`
  reports whether it's present.
- **Workspace root** — dispatched work lands under `.core/workspace/<org>/<repo>/task-<N>`,
  with a `db.duckdb` of permanent dispatch stats (`agentic:workspace/stats`).
- `core-agent check` is the fastest "is this install wired correctly?" probe.

## Next

[dispatch](../dispatch/) · [pipeline](../pipeline/) · [plans](../plans/) ·
[fleet](../fleet/) · [brain](../brain/) · [inference](../inference/) ·
[setup](../setup/) · [shell](../shell/) — and [`../architecture.md`](../architecture.md)
for how the packages fit together.
