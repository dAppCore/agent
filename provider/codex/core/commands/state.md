---
name: state
description: Read and write shared plan state for cross-session CoreAgent work
args: "[set|get|list|delete] <plan> [options]"
---

# Shared Plan State

Use state when a plan needs durable key/value context across sessions or agent handoffs.

## CLI Fallback

```bash
core-agent state set <plan> --key=pattern --value=observer --type=general
core-agent state get <plan> --key=pattern
core-agent state list <plan>
core-agent state delete <plan> --key=pattern
```

## Behaviour

Store facts that future agents should rely on: architectural decisions, API contracts, known blockers, verified commands, and chosen conventions. Do not store secrets or large logs.
