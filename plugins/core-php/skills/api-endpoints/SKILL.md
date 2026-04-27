---
name: api-endpoints
description: Use when a Core plugin needs to call api.lthn.sh or mcp.lthn.sh. Documents the JSON headers, auth, and endpoint conventions required by the plugin restructure RFC.
---

# API and MCP Endpoints

Use this skill when a command, script, or tool talks to the shared Core endpoints.

## Canonical endpoints

- `https://api.lthn.sh` — REST API
- `https://mcp.lthn.sh` — MCP bridge endpoint

These conventions apply to both production and self-hosted `lthn.sh` installs.

## Required headers

- Send `Accept: application/json` on every API request. The default response may be HTML if this header is missing.
- Send `Content-Type: application/json` for JSON request bodies.
- Send `Authorization: Bearer <token>` for authenticated requests.

## REST shape

- Use `/v1/{resource}` paths.
- Expect JSON request and response bodies.
- Keep request payloads explicit and small.

Example:

```bash
curl -s https://api.lthn.sh/v1/issues \
  -H 'Accept: application/json' \
  -H "Authorization: Bearer ${TOKEN}"
```

```bash
curl -s https://api.lthn.sh/v1/brain/recall \
  -X POST \
  -H 'Accept: application/json' \
  -H 'Content-Type: application/json' \
  -H "Authorization: Bearer ${TOKEN}" \
  -d '{"query":"dispatch status","top_k":3}'
```

## MCP usage

- Remote MCP can be reached at `https://mcp.lthn.sh`.
- Local plugin configs in this repository use `core mcp serve` through `.mcp.json`.

Local `.mcp.json` pattern:

```json
{
  "mcpServers": {
    "core": {
      "type": "stdio",
      "command": "core",
      "args": ["mcp", "serve"]
    }
  }
}
```
