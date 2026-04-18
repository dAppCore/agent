---
name: perf
description: Performance profiling helpers for Go and PHP.
args: <subcommand> [options]
---

# Performance Profiling

Profile test suite, HTTP requests, and analyze slow queries and memory usage.

## Subcommands

- `test`: Profile the test suite.
- `request <url>`: Profile an HTTP request.
- `query <query>`: Analyze a slow query (requires MySQL client and credentials).
- `memory [script_path]`: Analyze memory usage.

## Usage

```
/core:perf test
/core:perf request /api/users
/core:perf query "SELECT * FROM users WHERE email = 'test@example.com'"
/core:perf memory
```

## Actions

### Test Profiling

Run this command to profile the test suite:

```bash
"${CLAUDE_PLUGIN_ROOT}/scripts/perf-test.sh"
```

### Request Profiling

Run this command to profile an HTTP request:

```bash
"${CLAUDE_PLUGIN_ROOT}/scripts/perf-request.sh" "<url>"
```

### Query Analysis

Run this command to analyze a slow query:

```bash
"${CLAUDE_PLUGIN_ROOT}/scripts/perf-query.sh" "<query>"
```

### Memory Analysis

Run this command to analyze memory usage:

```bash
"${CLAUDE_PLUGIN_ROOT}/scripts/perf-memory.sh" "<script_path>"
```
