---
name: clean
description: Clean up generated files, caches, and build artifacts
args: "[--cache] [--deps [--force]] [--dry-run]"
---

# Clean Project

Cleans up generated files, caches, and build artifacts for the project.

## Usage

- `/core:clean` - Clean all caches and build artifacts.
- `/core:clean --cache` - Clean caches only.
- `/core:clean --deps` - Dry-run dependency cleanup.
- `/core:clean --deps --force` - **Permanently delete** dependencies (`vendor`, `node_modules`).
- `/core:clean --dry-run` - Show what would be deleted without actually deleting anything.

## Action

This command executes the `clean.sh` script to perform the cleanup.

```bash
"${CLAUDE_PLUGIN_ROOT}/scripts/clean.sh" "$@"
```
