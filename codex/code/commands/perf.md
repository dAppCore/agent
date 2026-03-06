---
name: perf
description: Performance profiling helpers for Go and PHP
args: <subcommand> [options]
---

# Performance Profiling

A collection of helpers to diagnose performance issues.

## Usage

Profile the test suite:
`/core:perf test`

Profile an HTTP request:
`/core:perf request /api/users`

Analyse slow queries:
`/core:perf query`

Analyse memory usage:
`/core:perf memory`

## Action

This command delegates to a shell script to perform the analysis.

```bash
/bin/bash "${CLAUDE_PLUGIN_ROOT}/scripts/perf.sh" "<subcommand>" "<options>"
```
