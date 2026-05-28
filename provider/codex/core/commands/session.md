---
name: session
description: Manage persistent CoreAgent sessions, handoffs, logs, artifacts, replay, and resume context
args: "[start|get|list|continue|handoff|end|complete|log|artifact|resume|replay] [options]"
---

# Sessions

Use sessions when work needs continuity across agents, runs, pauses, or handoffs. Sessions keep plan context, work logs, artifact history, and replayable state.

## CLI Fallback

```bash
core-agent session start <plan-slug> --agent-type=claude:opus
core-agent session list --plan=<plan-slug> --status=active
core-agent session get <session-id>
core-agent session continue <session-id> --agent-type=codex --work-log='[{"type":"checkpoint","message":"..."}]'
core-agent session log <session-id> --message="Checked build" --type=checkpoint
core-agent session artifact <session-id> --path="pkg/agentic/session.go" --action=modified
core-agent session handoff <session-id> --summary="Ready for review" --next-steps="Run verifier"
core-agent session end <session-id> --summary="Complete" --status=completed
core-agent session resume <session-id>
core-agent session replay <session-id>
```

## Behaviour

- Use `session log` for meaningful progress, blockers, and verification results.
- Use `session artifact` for created, modified, deleted, or reviewed files.
- Use `handoff` before changing agents or pausing work.
- Use `replay` to rebuild concise context before resuming long-running work.
