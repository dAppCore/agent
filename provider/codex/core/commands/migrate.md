---
name: migrate
description: Laravel migration helpers
args: <subcommand> [options]
---

# Laravel Migration Helper

Commands to help with Laravel migrations in the monorepo.

## Usage

`/core:migrate create <name> [--path <path>]` - Create a new migration file.
`/core:migrate run` - Run pending migrations.
`/core:migrate rollback` - Rollback the last database migration.
`/core:migrate fresh` - Drop all tables and re-run all migrations.
`/core:migrate status` - Show the status of each migration.
`/core:migrate from-model <model> [--model-path <path>] [--path <path>]` - Generate a migration from a model (experimental).

## Actions

### Create

Run this command to create a new migration:

```bash
"${CLAUDE_PLUGIN_ROOT}/scripts/create.sh" "<name>" "--path" "<path>"
```

### Run

Run this command to run pending migrations:

```bash
"${CLAUDE_PLUGIN_ROOT}/scripts/run.sh"
```

### Rollback

Run this command to rollback the last migration:

```bash
"${CLAUDE_PLUGIN_ROOT}/scripts/rollback.sh"
```

### Fresh

Run this command to drop all tables and re-run migrations:

```bash
"${CLAUDE_PLUGIN_ROOT}/scripts/fresh.sh"
```

### Status

Run this command to check migration status:

```bash
"${CLAUDE_PLUGIN_ROOT}/scripts/status.sh"
```

### From Model

Run this command to generate a migration from a model:

```bash
"${CLAUDE_PLUGIN_ROOT}/scripts/from-model.sh" "<model>" "--model-path" "<model-path>" "--path" "<path>"
```
