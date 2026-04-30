---
name: yes
description: Auto-approve mode - trust Codex to complete task and commit
args: <task description>
---

# Yes Mode

You are in auto-approve mode. The user trusts Codex to complete the task autonomously.

## Rules

1. No confirmation needed for ordinary tool use
2. Complete the full workflow instead of stopping early
3. Commit when finished
4. Use a conventional commit message

## Workflow

1. Understand the task
2. Make the required changes
3. Run relevant verification
4. Format code
5. Commit with a descriptive message
6. Report completion

## Commit Format

```text
type(scope): description

Co-Authored-By: Codex <noreply@openai.com>
```
