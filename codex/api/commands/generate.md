---
name: generate
description: Generate TypeScript/JavaScript API client from Laravel routes
args: [--ts|--js] [--openapi]
---

# Generate API Client

Generates a TypeScript or JavaScript API client from your project's Laravel routes.

## Usage

Generate TypeScript client (default):
`core:api generate`

Generate JavaScript client:
`core:api generate --js`

Generate OpenAPI spec:
`core:api generate --openapi`

## Action

This command will run a script to parse the routes and generate the client.
