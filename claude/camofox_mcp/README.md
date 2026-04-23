<!-- SPDX-License-Identifier: EUPL-1.2 -->

# camofox-mcp

`camofox-mcp` is a stdio MCP server that wraps the Camofox browser HTTP API for Claude Code.

## Install

Local editable install:

```bash
cd claude/camofox_mcp
pip install -e .
```

Direct git install:

```bash
pip install "git+https://github.com/dAppCore/core-agent.git#subdirectory=claude/camofox_mcp"
```

## Claude Code

```bash
claude mcp add camofox -- camofox-mcp --camofox-url=http://localhost:8099 --api-key=$CAMOFOX_API_KEY
```

If `--api-key` is omitted, the server will read `CAMOFOX_API_KEY` from the environment.

## Tools

- `navigate(url)` opens a new tab and returns `{tab_id, status}`
- `read_page(tab_id)` returns `{text, url, title}`
- `screenshot(tab_id)` returns `{image_b64}`
- `click(tab_id, selector)` returns `{ok}`
- `fill(tab_id, selector, value)` returns `{ok}`
- `close_tab(tab_id)` returns `{ok}`

The server prefers the official Python `mcp` SDK when it is importable. If that package is unavailable at runtime, it falls back to a small stdio JSON-RPC MCP implementation that supports `initialize`, `tools/list`, and `tools/call`.
