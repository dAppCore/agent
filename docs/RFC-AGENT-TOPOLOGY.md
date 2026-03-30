# Agent Fleet Topology

> How Cladius, Charon, and community agents are deployed, connected, and onboarded.

---

## Current Fleet

| Agent | Hardware | Location | Role |
|-------|----------|----------|------|
| Cladius | M3 Studio (36GB) | Local (Snider's desk) | Project leader, architecture, specs, dispatch |
| Charon | Ryzen 9 + 128GB + RX 7800 XT | Homelab (10.69.69.165) | Infrastructure, training, blockchain, DevOps |
| Codex agents | OpenAI cloud | Remote (sandboxed) | Implementation, polish, QA |
| Gemini agents | Google cloud | Remote | Research, analysis, alternative perspectives |

## Connectivity

```
Cladius (M3 Studio)
  └── core-agent MCP (stdio) → Claude Code
  └── agent_send → Charon (api.lthn.sh)

Charon (Homelab)
  └── core-agent MCP (stdio) → Claude Code
  └── agent_send → Cladius (api.lthn.sh)
  └── Ollama (local inference)
  └── Qdrant (OpenBrain vectors)

Both → OpenBrain (shared knowledge)
Both → Forge (git repos)
Both → api.lthn.sh / mcp.lthn.sh (MCP over HTTP)
```

## DNS Routing Strategy

Subdomains, not paths:
- `api.lthn.sh` — REST API
- `mcp.lthn.sh` — MCP endpoint
- `forge.lthn.ai` — Forgejo (de1 production)

Why subdomains: each service can have its own TLS cert, its own Traefik rule,
its own rate limiting. Paths create coupling.

## Community Onboarding (*.lthn.sh)

The `*.lthn.sh` wildcard resolves to 10.69.69.165 (homelab) for Snider,
but for community members it resolves to 127.0.0.1 (localhost).

This means:
1. Community member installs core-agent
2. core-agent starts local MCP server
3. `api.lthn.sh` resolves to their own localhost
4. They're running their own node — no dependency on Snider's hardware
5. When they're ready, they peer with the network via WireGuard

BugSETI bootstrap tool automates this: bare metal → running node in 10 steps.

## Fleet Dispatch (lthn.sh)

lthn.sh is the fleet controller:
1. Orchestrator creates task
2. Task assigned to agent pool (codex, gemini, claude, local)
3. Agent picks up via SSE/polling from api.lthn.sh
4. Runs in sandboxed workspace
5. Reports completion via checkin API
6. Orchestrator reviews, merges, or sends back

Community members contribute compute by running core-agent connected to the fleet.
