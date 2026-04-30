---
name: migrate
description: Manage Laravel migrations in the monorepo
args: <subcommand> [arguments]
---

# Laravel Migration Helper

Commands to help with Laravel migrations in the monorepo.

## Subcommands

### `create <name>`
Create a new migration file.
e.g., `/core:migrate create create_users_table`

### `run`
Run pending migrations.
e.g., `/core:migrate run`

### `rollback`
Rollback the last batch of migrations.
e.g., `/core:migrate rollback`

### `fresh`
Drop all tables and re-run all migrations.
e.g., `/core:migrate fresh`

### `status`
Show the migration status.
e.g., `/core:migrate status`

### `from-model <model>`
Generate a migration from a model.
e.g., `/core:migrate from-model User`
