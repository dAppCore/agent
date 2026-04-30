---
name: refactor
description: Guided refactoring with safety checks
args: <subcommand> [args]
---

# Refactor

Guided refactoring with safety checks.

## Subcommands

- `extract-method <new-method-name>` - Extract selection to a new method
- `rename <new-name>` - Rename a class, method, or variable
- `move <new-namespace>` - Move a class to a new namespace
- `inline` - Inline a method

## Usage

```
/core:refactor extract-method validateToken
/core:refactor rename User UserV2
/core:refactor move App\\Models\\User App\\Data\\Models\\User
/core:refactor inline calculateTotal
```

## Action

This command will run the refactoring script:

```bash
~/.claude/plugins/code/scripts/refactor.php "<subcommand>" [args]
```
