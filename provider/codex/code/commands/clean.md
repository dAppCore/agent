---
name: clean
description: Clean up generated files, caches, and build artifacts.
args: "[--deps] [--cache] [--dry-run]"
---

# Clean Project

This command cleans up generated files from the current project.

## Usage

```
/code:clean                   # Clean all
/code:clean --deps            # Remove vendor/node_modules
/code:clean --cache           # Clear caches only
/code:clean --dry-run         # Show what would be deleted
```

## Action

```bash
"${CLAUDE_PLUGIN_ROOT}/scripts/cleanup.sh" "$@"
```
