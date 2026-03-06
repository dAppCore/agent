---
name: api
description: Generate TypeScript/JavaScript API client from Laravel routes
args: generate [--ts|--js|--openapi]
---

# API Client Generator

Generate a TypeScript/JavaScript API client or an OpenAPI specification from your Laravel routes.

## Usage

Generate a TypeScript client (default):
`/code:api generate`
`/code:api generate --ts`

Generate a JavaScript client:
`/code:api generate --js`

Generate an OpenAPI specification:
`/code:api generate --openapi`

## Action

```bash
"${CLAUDE_PLUGIN_ROOT}/scripts/api-generate.sh" "$@"
```
