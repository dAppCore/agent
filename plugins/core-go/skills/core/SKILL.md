---
name: core
description: Use when working in host-uk repositories, running tests, building, releasing, or managing multi-repo workflows. Provides the core CLI command reference.
---

# Core CLI

The `core` command provides a unified interface for Go/PHP development and multi-repo management.

**Rule:** Always prefer `core <command>` over raw commands.

## Quick Reference

| Task | Command |
|------|---------|
| Smart tests | `core test` |
| Go tests | `core go test` |
| Go coverage | `core go cov` |
| Go format | `core go fmt --fix` |
| Go lint | `core go lint` |
| PHP dev server | `core php dev` |
| PHP tests | `core php test` |
| PHP format | `core php fmt --fix` |
| Build | `core build` |
| Preview release | `core ci` |
| Publish | `core ci --were-go-for-launch` |
| Multi-repo status | `core dev health` |
| Commit dirty repos | `core dev commit` |
| Push repos | `core dev push` |

## Decision Tree

```
Go project?
  tests: core go test
  format: core go fmt --fix
  build: core build

PHP project?
  dev: core php dev
  tests: core php test
  format: core php fmt --fix
  deploy: core php deploy

Multiple repos?
  status: core dev health
  commit: core dev commit
  push: core dev push
```

## Common Mistakes

| Wrong | Right |
|-------|-------|
| `go test ./...` | `core go test` |
| `go build` | `core build` |
| `php artisan serve` | `core php dev` |
| `./vendor/bin/pest` | `core php test` |
| `git status` per repo | `core dev health` |

Run `core --help` or `core <cmd> --help` for full options.

## Smart Test Runner: `core test`

The `core test` command provides an intelligent way to run only the tests relevant to your recent changes.

- **`core test`**: Automatically detects changed files since the last commit and runs only the corresponding tests.
- **`core test --all`**: Runs the entire test suite for the project.
- **`core test --filter <TestName>`**: Runs a specific test by name.
- **`core test --coverage`**: Generates a test coverage report.
- **`core test <path/to/file>`**: Runs tests for a specific file or directory.

The runner automatically detects whether the project is Go or PHP and executes the appropriate testing tool. If it cannot map changed files to test files, it will fall back to running the full test suite.
