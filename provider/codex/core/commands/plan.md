---
name: plan
description: Create, inspect, update, check, archive, and delete CoreAgent implementation plans
args: "[templates|create|from-issue|list|show|update|status|check|archive|delete] [options]"
---

# Plans

Use CoreAgent plans for multi-step implementation work, issue decomposition, phase checkpoints, and task-level progress tracking.

## Preferred Routing

Use MCP plan tools when available:
- `agentic_plan_create`
- `agentic_plan_read`
- `agentic_plan_update`
- `agentic_plan_delete`
- `agentic_plan_list`

Use CLI fallback:

```bash
core-agent plan templates --category=development
core-agent plan create <slug> --title="..." --objective="..." --import=bug-fix --activate
core-agent plan from-issue <slug> --id=N
core-agent plan list --status=ready --repo=go-io
core-agent plan show <slug>
core-agent plan update <slug> --status=ready --notes="..."
core-agent plan status <slug> --set=active
core-agent plan check <slug> --phase=1
core-agent plan archive <slug> --reason="superseded"
core-agent plan delete <id> --reason="created by mistake"
```

## Phase And Task Controls

Use these when the user asks for phase progress, task toggles, or checkpoints:

```bash
core-agent phase get <plan> --phase=1
core-agent phase update-status <plan> --phase=1 --status=completed --reason="verified"
core-agent phase add-checkpoint <plan> --phase=1 --note="Build passes"
core-agent task create <plan> --phase=1 --title="Patch runner coverage"
core-agent task update <plan> --phase=1 --task=1 --status=completed --notes="Done"
```

## Behaviour

For implementation work that spans several files or systems, create or update a plan before dispatching extra agents. Keep statuses evidence-based and include exact verification commands in checkpoints.
