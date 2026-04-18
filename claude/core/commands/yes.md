---
name: yes
description: Auto-approve mode - trust Claude to complete task and commit
args: <task description>
hooks:
  PermissionRequest:
    - hooks:
        - type: command
          command: "${CLAUDE_PLUGIN_ROOT}/scripts/auto-approve.sh"
  Stop:
    - hooks:
        - type: command
          command: "${CLAUDE_PLUGIN_ROOT}/scripts/ensure-commit.sh"
          once: true
---

# Yes Mode

You are in **auto-approve mode**. The user trusts you to complete this task autonomously.

## Task

$ARGUMENTS

## Rules

1. **No confirmation needed** - all tool uses are pre-approved
2. **Complete the full workflow** - don't stop until done
3. **Commit when finished** - create a commit with the changes
4. **Use conventional commits** - type(scope): description

## Workflow

1. Understand the task
2. Make necessary changes (edits, writes)
3. Run tests to verify (`core go test` or `core php test`)
4. Format code (`core go fmt` or `core php fmt`)
5. Commit changes with descriptive message
6. Report completion

Do NOT stop to ask for confirmation. Just do it.

## Commit Format

```
type(scope): description

Co-Authored-By: Claude Opus 4.5 <noreply@anthropic.com>
```

Types: feat, fix, refactor, docs, test, chore

## Safety Notes

- The Stop hook will block if you try to stop with uncommitted changes
- You still cannot bypass blocked commands (security remains enforced)
- If you get stuck in a loop, the user can interrupt with Ctrl+C
