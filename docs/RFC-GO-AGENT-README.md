# core/agent — Agentic Orchestration

`dappco.re/go/agent` — The agent dispatch, monitoring, and fleet management system.

## Status

- **Version:** v0.10.0-alpha.1
- **RFC:** `code/core/agent/docs/RFC.md` + `code/core/agent/docs/RFC.plan.md`
- **Tests:** 8 packages, all passing
- **Binary:** `core-agent` (MCP server + CLI)

## What It Does

core-agent is both a binary (`core-agent`) and a library. It provides:

- **MCP server** — stdio transport, tool registration, channel notifications
- **Dispatch** — prep workspaces, spawn codex/claude/gemini agents in Docker
- **Runner service** — concurrency limits, queue drain, frozen state
- **Monitor** — background check loop, completion detection, inbox polling
- **Brain** — OpenBrain integration (recall, remember, forget)
- **Messaging** — agent-to-agent messages via lthn.sh API

## Architecture

```
cmd/core-agent/main.go
  ├── agentic.Register     ← workspace prep, dispatch, MCP tools
  ├── runner.Register      ← concurrency, queue drain, frozen state
  ├── monitor.Register     ← background checks, channel notifications
  ├── brain.Register       ← OpenBrain tools
  └── mcp.Register         ← MCP server + ChannelPush
```

Services communicate via Core IPC:
- `AgentStarted` → runner pushes ChannelPush → MCP sends to Claude Code
- `AgentCompleted` → runner updates Registry + pokes queue + ChannelPush
- `ChannelPush` → MCP HandleIPCEvents → ChannelSend to stdout
