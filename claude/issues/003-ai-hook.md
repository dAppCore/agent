# feat(ai): Add hook validation for Claude Code PreToolUse

## Summary

Add `core ai hook` subcommands to validate commands and file operations for Claude Code hooks.

## Required Commands

```bash
core ai hook validate-command <cmd>   # Check if command is safe/allowed
core ai hook validate-file <path>     # Check if file creation is allowed
core ai hook post-commit              # Check for uncommitted work after commit
```

## Current Shell Scripts Being Replaced

- `claude/hooks/prefer-core.sh` - Blocks dangerous commands, enforces core CLI
- `claude/scripts/block-docs.sh` - Blocks random .md file creation
- `claude/scripts/post-commit-check.sh` - Warns about uncommitted work

## Command Validation Rules

Block these patterns:
- `rm -rf` / `rm -r` (except node_modules, vendor, .cache, dist, build)
- `mv`/`cp` with wildcards
- `xargs` with rm/mv/cp
- `find -exec` with file operations
- `sed -i` (in-place editing)
- `grep -l | ...` (mass file targeting)
- `perl -i`, `awk > file`

Redirect to core:
- `go test/build/fmt/mod` → suggest `core go *`
- `golangci-lint` → suggest `core go lint`
- `composer test` → suggest `core php test`
- `./vendor/bin/pint` → suggest `core php fmt`
- `php artisan serve` → suggest `core php dev`

## File Validation Rules

Allow:
- `README.md`, `CLAUDE.md`, `AGENTS.md`, `CONTRIBUTING.md`, `CHANGELOG.md`, `LICENSE.md`
- Files in `docs/` directory

Block:
- Other `.md` files (suggest using README.md or docs/)

## Output Format (JSON for hooks)

```json
{"decision": "approve"}
```

```json
{"decision": "block", "message": "Use `core go test` instead of raw go test"}
```

```json
{"decision": "warn", "message": "3 files remain uncommitted after commit"}
```
