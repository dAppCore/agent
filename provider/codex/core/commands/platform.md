---
name: platform
description: Manage Core platform sync, auth, fleet nodes, fleet tasks, credits, subscriptions, and agent messages
args: "[sync|auth|login|fleet|credits|subscription|message] [options]"
---

# Platform Integration

Use this family for multi-agent platform state, fleet coordination, authentication, credits, subscriptions, and direct agent messages.

## Sync

```bash
core-agent sync push
core-agent sync pull <agent>
core-agent sync status
```

## Auth

```bash
core-agent login <6-digit-code>
core-agent auth provision <oauth-user-id> --name=codex --permissions=plans:read,plans:write
core-agent auth revoke <key-id>
```

## Fleet

```bash
core-agent fleet register <agent-id> --platform=linux --models=codex,gpt-5.4
core-agent fleet heartbeat <agent-id>
core-agent fleet nodes
core-agent fleet events
core-agent fleet task next
core-agent fleet task assign --node=<agent-id> --task='{"repo":"go-io"}'
core-agent fleet task complete --task-id=<id> --status=completed
core-agent fleet stats
core-agent fleet deregister <agent-id>
```

## Credits And Subscription

```bash
core-agent credits balance <agent-id>
core-agent credits history <agent-id>
core-agent credits award <agent-id> --amount=10 --reason="review"
core-agent subscription detect
core-agent subscription budget <agent-id>
core-agent subscription budget update <agent-id> --limit=100
```

## Messages

```bash
core-agent message send <workspace> --from=codex --to=claude --subject="Review" --content="Please check the prompt."
core-agent message inbox <workspace> --agent=claude
core-agent message conversation <workspace> --agent=codex --with=claude
```

Never print API keys or pairing secrets into chat. Summarise auth outcomes by key ID or prefix only.
