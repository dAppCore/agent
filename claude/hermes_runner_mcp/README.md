<!-- SPDX-License-Identifier: EUPL-1.2 -->

# hermes-runner-mcp

MCP stdio server that lets Claude Code dispatch sandboxed Hermes-runner jobs through a Hermes gateway.

## Install

```bash
pip install -e .
```

## Claude Code

```bash
claude mcp add hermes-runner -- hermes-runner-mcp --hermes-url=http://localhost:8642 --api-key=$HERMES_API_KEY
```

## Configuration

- `--hermes-url`: Hermes gateway base URL. Defaults to `http://localhost:8642/`.
- `--api-key`: Hermes gateway API key. Falls back to `HERMES_API_KEY`.

## Tools

- `hermes_dispatch(task, inputs, agents=None) -> {run_id, status_url}`
- `hermes_status(run_id) -> {state, progress, last_event}`
- `hermes_fetch(run_id) -> {output, artifacts, log}`

If the `mcp` Python SDK is installed, the server uses FastMCP. If not, it falls back to a newline-delimited JSON-RPC stdio implementation compatible with the MCP stdio transport.

The gateway client expects the primary routes below and retries a small set of conventional fallbacks if the primary route returns `404`:

- `POST /dispatch`
- `GET /runs/{run_id}`
- `GET /runs/{run_id}/fetch`

When `agents` is provided to `hermes_dispatch`, the request body includes both the raw `agents` list and `args: ["--agents", "<json>"]` so the remote runner can preserve Hermes subagent composition.
