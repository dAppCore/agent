<!-- SPDX-License-Identifier: EUPL-1.2 -->
# CLI & getting started

**core-agent** is a single Go binary that runs both as an **MCP server** (driven by IDEs
and other agents) and as a **command-line tool** for orchestrating AI coding agents
across the Core ecosystem. This page covers building it and its run modes; the full
command list is in [commands](commands.md).

## Build & install

```bash
cd go
go build ./cmd/core-agent/      # → ./core-agent
go install ./cmd/core-agent/    # → $GOPATH/bin
```

Cross-compile for the homelab Linux box (Charon):

```bash
cd go && GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -o core-agent-linux ./cmd/core-agent/
```

The binary is **dual-named**: invoked as `core-agent` it is the legacy default; installed
or symlinked as `lthn-agent` it identifies as part of the `lthn-{mlx,cuda,amd,agent}`
family (`main.go:detectBinaryName`). Same behaviour, different identity in banners and
admin-token prefixes.

## Run modes

| Command | Transport | For |
|---------|-----------|-----|
| `core-agent mcp` | MCP over **stdio** | IDE integration (Claude Code etc.) |
| `core-agent serve` | MCP over **HTTP** | cross-agent comms, CI, the fleet |
| `core-agent hub` | loopback HTTP + MCP HTTP/SSE | the agent control plane (opencode + brain) |

`mcp`/`serve` come from the shared `dappco.re/go/mcp` service; everything else is
registered by `cmd/core-agent` (`commands.go`).

## Configuration

- **`agents.yaml`** — fleet + agent config (`agentic.AgentsConfigPath()`).
- **Workspace root** — dispatched work lands under `.core/workspace/<org>/<repo>/task-<N>`.
- `core-agent check` verifies the install; `core-agent version` / `env` report build +
  environment.

## In this section

- [commands](commands.md) — the full command reference (chat, engine control, dispatch
  verbs, maintenance).

**Related:** [dispatch](../dispatch/) · [inference](../inference/) · [shell](../shell/) ·
[fleet](../fleet/) · [architecture](../architecture.md).
