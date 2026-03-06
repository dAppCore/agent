---
name: todo
description: Extract and track TODOs from the codebase
args: '[add "message" | done <id> | --priority]'
---

# TODO Command

This command scans the codebase for `TODO`, `FIXME`, `HACK`, and `XXX` comments and displays them in a formatted list.

## Usage

List all TODOs:
`/core:todo`

Sort by priority:
`/core:todo --priority`

## Action

```bash
"${CLAUDE_PLUGIN_ROOT}/scripts/todo.sh" <args>
```
