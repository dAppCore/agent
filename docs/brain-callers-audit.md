<!-- SPDX-License-Identifier: EUPL-1.2 -->

# Brain Callers Audit

Date: 2026-04-25  
Ticket: Mantis #121

## Scope

Audit command:

```bash
rg -n '/v1/brain' /Users/snider/Code/core/agent /Users/snider/Code/core/mcp
```

Tests, PHP/Laravel handlers, and documentation-only references were excluded when classifying runtime callers.

## Verdict

This ticket is **not stale-fixed**.

- `core/agent` still had direct Go callers that bypassed the shared OpenBrain helper path. Those are patched in this ticket.
- `core/mcp` already has a hardened shared client and direct subsystem, but one MCP prep caller still bypasses that client.
- Hermes Python plugins and Claude shell hooks still call `/v1/brain/*` directly without a circuit-breaker or retry policy.
- `plugins/core-go/skills/api-endpoints/SKILL.md` is documentation only, not a runtime caller, but its example still shows the raw endpoint shape rather than the hardened client path.

## Hardened Baseline

The current non-Laravel baseline is the shared Go client in [client.go](/Users/snider/Code/core/mcp/pkg/mcp/brain/client/client.go:65):

- [client.go](/Users/snider/Code/core/mcp/pkg/mcp/brain/client/client.go:265) injects default org and agent on typed `Remember`, `Recall`, and `List` requests.
- [client.go](/Users/snider/Code/core/mcp/pkg/mcp/brain/client/client.go:310) routes requests through retry and circuit-breaker policy.
- [client.go](/Users/snider/Code/core/mcp/pkg/mcp/brain/client/client.go:504) opens and cools down the circuit.
- [client.go](/Users/snider/Code/core/mcp/pkg/mcp/brain/client/client.go:581) retries `408`, `429`, and `5xx`, with `Retry-After` support at [client.go](/Users/snider/Code/core/mcp/pkg/mcp/brain/client/client.go:585).

## Runtime Callers

| Path | Status | Org scope | Breaker / retry | Notes |
| --- | --- | --- | --- | --- |
| [pkg/brain/direct.go](/Users/snider/Code/core/agent/pkg/brain/direct.go:106) | patched | now defaults `org` from `CORE_BRAIN_ORG` when omitted | already used shared client `Call()` | Active `core-agent` brain subsystem |
| [pkg/agentic/prep.go](/Users/snider/Code/core/agent/pkg/agentic/prep.go:1200) via [pkg/agentic/brain_client.go](/Users/snider/Code/core/agent/pkg/agentic/brain_client.go:17) | patched | helper injects configured org when caller omitted it | helper now uses shared client + shared circuit breaker | Replaced raw `HTTPPost` recall |
| [pkg/agentic/session.go](/Users/snider/Code/core/agent/pkg/agentic/session.go:826) via [pkg/agentic/brain_client.go](/Users/snider/Code/core/agent/pkg/agentic/brain_client.go:17) | patched | helper injects configured org when caller omitted it | helper now uses shared client + shared circuit breaker | Replaced raw `HTTPPost` remember |
| [pkg/agentic/brain_seed_memory.go](/Users/snider/Code/core/agent/pkg/agentic/brain_seed_memory.go:153) via [pkg/agentic/brain_client.go](/Users/snider/Code/core/agent/pkg/agentic/brain_client.go:17) | patched | helper injects configured org when caller omitted it | helper now uses shared client + shared circuit breaker | Replaced raw `HTTPPost` remember while preserving `workspace_id` |
| [pkg/mcp/brain/direct.go](/Users/snider/Code/core/mcp/pkg/mcp/brain/direct.go:98) | aligned | typed client path carries org defaulting | shared client | Already on hardened path |
| [cmd/brain-seed/main.go](/Users/snider/Code/core/mcp/cmd/brain-seed/main.go:67) and [cmd/brain-seed/main.go](/Users/snider/Code/core/mcp/cmd/brain-seed/main.go:257) | aligned | org passed into shared client and request input | shared client | Already on hardened path |
| [pkg/mcp/agentic/prep.go](/Users/snider/Code/core/mcp/pkg/mcp/agentic/prep.go:641) | follow-up | no explicit org in request body | raw `http.NewRequest` + `s.client.Do`, no shared breaker / retry | Read-only in this sandbox; should be switched to `pkg/mcp/brain/client` |
| [hermes/plugins/openbrain_memory.py](/Users/snider/Code/core/agent/hermes/plugins/openbrain_memory.py:284) and [hermes/plugins/openbrain_memory.py](/Users/snider/Code/core/agent/hermes/plugins/openbrain_memory.py:493) | follow-up | org is optional / caller-provided | direct `requests` / `httpx` / `urllib`, no breaker / retry | Outside allowed edit scope for this ticket |
| [hermes/plugins/openbrain_context.py](/Users/snider/Code/core/agent/hermes/plugins/openbrain_context.py:193) and [hermes/plugins/openbrain_context.py](/Users/snider/Code/core/agent/hermes/plugins/openbrain_context.py:526) | follow-up | org is optional / caller-provided | direct `requests` / `httpx` / `urllib`, no breaker / retry | Outside allowed edit scope for this ticket |
| [claude/core/scripts/session-start.sh](/Users/snider/Code/core/agent/claude/core/scripts/session-start.sh:20), [claude/core/scripts/session-save.sh](/Users/snider/Code/core/agent/claude/core/scripts/session-save.sh:57), [claude/core/scripts/pre-compact.sh](/Users/snider/Code/core/agent/claude/core/scripts/pre-compact.sh:74) | follow-up | no org field sent | raw `curl`, no breaker / retry | Outside the shell-script allowlist for this ticket |

## Documentation-Only Reference

- [plugins/core-go/skills/api-endpoints/SKILL.md](/Users/snider/Code/core/agent/plugins/core-go/skills/api-endpoints/SKILL.md:37) is not a runtime caller. It is still worth tightening so plugin authors are pointed at the shared client pattern or at least warned that raw `curl` examples omit org and breaker/retry policy.

## Changes Applied

- Added [pkg/agentic/brain_client.go](/Users/snider/Code/core/agent/pkg/agentic/brain_client.go:1) to centralise non-tool OpenBrain calls in `core-agent` onto the shared client with a subsystem-scoped circuit breaker and org injection.
- Updated [pkg/agentic/prep.go](/Users/snider/Code/core/agent/pkg/agentic/prep.go:1200), [pkg/agentic/session.go](/Users/snider/Code/core/agent/pkg/agentic/session.go:826), and [pkg/agentic/brain_seed_memory.go](/Users/snider/Code/core/agent/pkg/agentic/brain_seed_memory.go:153) to use that helper instead of raw `HTTPPost`.
- Updated [pkg/brain/direct.go](/Users/snider/Code/core/agent/pkg/brain/direct.go:106) so remember / recall / list send the configured org by default when callers omit it.

## Recommended Follow-Up

1. Patch [pkg/mcp/agentic/prep.go](/Users/snider/Code/core/mcp/pkg/mcp/agentic/prep.go:641) to use `pkg/mcp/brain/client`.
2. Patch Hermes OpenBrain plugins to reuse a shared client wrapper with org defaults plus retry / breaker logic.
3. Patch Claude shell hooks or retire them in favour of a small Go helper that uses the shared client.
4. Tighten [plugins/core-go/skills/api-endpoints/SKILL.md](/Users/snider/Code/core/agent/plugins/core-go/skills/api-endpoints/SKILL.md:37) so the example does not become a copy-paste bypass.

## Notes

- No top-level `scripts/*.sh` file in this repository currently calls `/v1/brain/*`.
- `/Users/snider/Code/core/mcp` was readable but not writable in this session, so the MCP prep caller could be audited but not patched here.
