# Core CLI MCP Server

This directory contains an MCP server that exposes the core CLI commands as tools for AI agents.

## Tools

### `core_go_test`

Run Go tests.

**Parameters:**

- `filter` (string, optional): Filter tests by name.
- `coverage` (boolean, optional): Enable code coverage. Defaults to `false`.

**Example:**

```json
{
  "tool": "core_go_test",
  "parameters": {
    "filter": "TestMyFunction",
    "coverage": true
  }
}
```

### `core_dev_health`

Check the health of the monorepo.

**Parameters:**

None.

**Example:**

```json
{
  "tool": "core_dev_health",
  "parameters": {}
}
```

### `core_dev_commit`

Commit changes across repositories.

**Parameters:**

- `message` (string, required): The commit message.
- `repos` (array of strings, optional): A list of repositories to commit to.

**Example:**

```json
{
  "tool": "core_dev_commit",
  "parameters": {
    "message": "feat: Implement new feature",
    "repos": [
      "core-agent",
      "another-repo"
    ]
  }
}
```
