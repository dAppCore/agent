# feat(skill): Add /core:yes auto-approve and commit skill

## Summary

Add a `/core:yes` skill that auto-approves all permission requests and ensures work completes with a commit - like a "just do it" mode for trusted workflows.

## Use Case

When you trust Claude to complete a task end-to-end:
```
/core:yes fix the failing test and commit
```

Instead of:
1. Claude proposes edit → you approve
2. Claude runs test → you approve
3. Claude proposes commit → you approve

It just does it all and commits.

## Skill Definition

`claude/commands/yes.md`:

```markdown
---
name: yes
description: Auto-approve mode - trust Claude to complete task and commit
args: <task description>
hooks:
  PermissionRequest:
    - hooks:
        - type: command
          command: "core ai auto-approve"
  Stop:
    - hooks:
        - type: command
          command: "core ai ensure-commit"
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
```

## Hook Implementations

### core ai auto-approve

Auto-approves all permission requests during the skill:

```bash
#!/bin/bash
# Reads PermissionRequest input, returns allow decision

INPUT=$(cat)
TOOL=$(echo "$INPUT" | jq -r '.tool_name')

# Log what we're approving
echo "[yes-mode] Auto-approving: $TOOL" >&2

# Return allow decision
cat << EOF
{
  "hookSpecificOutput": {
    "hookEventName": "PermissionRequest",
    "decision": {
      "behavior": "allow"
    }
  }
}
EOF
```

### core ai ensure-commit

Stop hook that ensures work ends with a commit:

```bash
#!/bin/bash
INPUT=$(cat)
STOP_ACTIVE=$(echo "$INPUT" | jq -r '.stop_hook_active')

# Prevent infinite loop
if [ "$STOP_ACTIVE" = "true" ]; then
  exit 0
fi

# Check for uncommitted changes
CHANGES=$(git status --porcelain 2>/dev/null | wc -l | tr -d ' ')

if [ "$CHANGES" -gt 0 ]; then
  cat << EOF
{
  "decision": "block",
  "reason": "You have uncommitted changes. Please commit them before stopping. Use: git add -A && git commit -m 'type(scope): description'"
}
EOF
else
  exit 0
fi
```

## Safety Considerations

1. **Skill-scoped hooks** - only active while `/core:yes` is running
2. **once: true on Stop hook** - prevents infinite loops
3. **Still respects blocked commands** - prefer-core.sh still blocks dangerous patterns
4. **Commit required** - ensures changes are tracked

## Alternative: Permission Mode

Could also be implemented as a permission mode flag:

```bash
claude --permission-mode=dontAsk
```

But the skill approach is more explicit and adds the commit requirement.

## Example Usage

```
> /core:yes add a --version flag to the CLI that prints the version from go.mod

[yes-mode] Auto-approving: Read
[yes-mode] Auto-approving: Edit
[yes-mode] Auto-approving: Bash (core go test)
[yes-mode] Auto-approving: Bash (git commit)

Done. Created commit: feat(cli): add --version flag
```

## Integration with core CLI

Eventually this becomes:

```bash
core ai yes "fix the failing test"
```

Which spawns Claude in yes-mode with the task.
