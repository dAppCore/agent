---
name: sync
description: Sync changes across dependent modules
args: <module_name> [--dry-run]
---

# Sync Dependent Modules

When changing a base module, this command syncs the dependent modules.

## Usage

```
/code:sync                # Sync all dependents of current module
/code:sync core-tenant    # Sync specific module
/code:sync --dry-run      # Show what would change
```

## Action

```bash
"${CLAUDE_PLUGIN_ROOT}/scripts/sync.sh" "$@"
```
