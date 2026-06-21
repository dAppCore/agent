<!-- SPDX-Licence-Identifier: EUPL-1.2 -->

# Design — `provider/opencode` plugin (v1)

**Date:** 2026-05-31 · **Author:** Cladius (Opus) · **Status:** awaiting user review
**Decisions (brainstorming):** bridge = **HTTP to the hub** (the loopback control plane, Mantis
#1807) · v1 scope = **core capability subset (`dispatch`, `status`, `scan`, `brain_recall`,
`brain_remember`) as `tool()` exports + the report-home lifecycle hooks**.

## Context

RFC §7 promises a `provider/opencode/` plugin (`@opencode-ai/plugin`) — the opencode twin of
`provider/claude/`. A clean survey confirmed it **does not exist** anywhere in the repo
(`go/pkg/opencode` is the Go-side *host* — Service/Generate/proxy/control/hub — not the JS plugin;
`provider/` holds claude, codex, google, hermes only). Separately, RFC §12 flags the **report-home
loop** as "out of action." This plugin is the missing opencode side of both: it exposes
core-agent's capabilities to a running opencode session **and** reports that session's progress
home so the orchestrator (Cladius) sees it.

The bridge is **HTTP to the hub** — the loopback control plane the hub mode already serves and
that RFC §2 calls "the surface the plugins drive." This is the sandbox-correct choice: a dispatched
opencode session runs in a container and may not have the `core-agent` binary on PATH, but it can
always reach the host's loopback hub.

## Goal

A working, tested `@opencode-ai/plugin` at `provider/opencode/` that, when loaded by any opencode
instance:
1. exposes `dispatch`, `status`, `scan`, `brain_recall`, `brain_remember` as custom `tool()`s the
   model can call, each bridged to the hub over HTTP;
2. reports session lifecycle home (`session.idle` → done, `session.error` → BLOCKED,
   `tool.execute.after` → throttled progress) by calling the hub's `agent_send`;
3. **never breaks the session** — every hub call is failure-isolated (a hub that is down, a missing
   token, a non-2xx, a thrown error all degrade to a returned error string for tools / a silent
   no-op for hooks).

## Transport — the hub plane

The hub serves two loopback planes (Mantis #1807, `commands_hub.go`):
- **`:9201`** — bearer-auth REST control plane (`coreapi.Engine`): opencode control
  (`/v1/api/opencode`), sandbox proxy (`/v1/api/sandbox`), brain (`/api/brain/{remember,recall,
  forget,list,status}`).
- **`:9202`** — fail-closed MCP HTTP+SSE tool plane (`POST /mcp` JSON-RPC 2.0 `tools/call`; `GET
  /mcp` SSE), per-request bearer, requires `MCP_JWT_SECRET`.

**The chosen plane is `:9202`, via its stateless REST bridge** — confirmed in
`external/mcp/go/pkg/mcp/transport_http.go` + `bridge_api.go`. `ServeHTTP` auto-mounts every MCP
tool as a plain REST endpoint at **`POST /v1/tools/<tool_name>`** alongside the JSON-RPC `/mcp`
endpoint. The bridge binds the JSON request body directly as the tool's arguments
(`ShouldBindJSON(&payload map[string]any)`) and writes the tool result as JSON — **no JSON-RPC
envelope, no `initialize`, no `Mcp-Session-Id` handshake.** This carries all five tools +
`agent_send` (verified registered: `agentic_dispatch`, `agentic_status`, `agentic_scan`,
`brain_recall`, `brain_remember`, `agent_send`). The `:9201` REST plane carries only
brain + opencode-control (not dispatch/status/scan), so it cannot serve v1; we use `:9202`'s bridge.

**Wire shape (confirmed):** `POST {base}/v1/tools/<tool_name>`, header `Authorization: Bearer
<token>`, `Content-Type: application/json`, body = the arguments object (e.g.
`{"repo":"r","task":"t"}`). Response = the tool output as JSON. (The JSON-RPC `POST /mcp`
`tools/call` path remains a documented fallback behind the same `HubClient` interface if the bridge
proves insufficient — but the bridge is the v1 default.)

**Auth (O3, resolved):** the bearer is the hub's **`MCP_AUTH_TOKEN`** (the per-request secret
`withAuth` checks; a JWT can alternatively be exchanged at `POST /mcp/auth`, not needed for v1). The
plugin's `CORE_HUB_TOKEN` therefore carries the `MCP_AUTH_TOKEN` value.

**Config (env, read once at plugin init):**
- `CORE_HUB_URL` — base, default `http://127.0.0.1:9202`.
- `CORE_HUB_TOKEN` / `CORE_HUB_TOKEN_FILE` — the bearer = the hub's `MCP_AUTH_TOKEN`. If neither is
  set, tools return a clear "hub token not configured" string and hooks no-op.
- `CORE_REPORT_TO` — report-home target agent, default `cladius`.
- `CORE_REPORT_WORKSPACE` — the workspace id `agent_send` requires (see Open question O1).
- `CORE_PROGRESS_INTERVAL_MS` — progress throttle, default `60000`.

A `HubClient` interface wraps the transport so (a) the plane is a one-line default, not baked into
every tool, and (b) tests inject a fake client with **no network**.

## What it is

```typescript
import { type Plugin, tool } from "@opencode-ai/plugin"

export const CoreAgent: Plugin = async (ctx) => {
  // ctx: { project, directory, worktree, client, $ }
  const cfg  = loadConfig(process.env)         // pure
  const hub  = makeHubClient(cfg)              // HubClient (real fetch transport)
  return {
    tool: {
      dispatch:        dispatchTool(hub),
      status:          statusTool(hub),
      scan:            scanTool(hub),
      brain_recall:    brainRecallTool(hub),
      brain_remember:  brainRememberTool(hub),
    },
    event: async ({ event }) => reportLifecycle(hub, cfg, event),  // idle/error
    "tool.execute.after": async (input) => reportProgress(hub, cfg, input),
  }
}
```

## Tool surface (v1)

Each `tool()` validates args with `tool.schema.*` (Zod), calls one hub MCP tool, returns the text
result. Names are the opencode-facing names; the hub MCP tool names are fixed.

| opencode tool | args (v1) | hub MCP tool |
|---|---|---|
| `dispatch` | `repo, task, agent?, issue?` | `agentic_dispatch` |
| `status` | `workspace?` | `agentic_status` |
| `scan` | `org?, repo?` | `agentic_scan` |
| `brain_recall` | `query, limit?` | `brain_recall` |
| `brain_remember` | `content, tags?` | `brain_remember` |

Exact arg keys are reconciled against each hub tool's input schema in plan Task 1's spike (the spike
dumps `tools/list`), so the typed schemas match the server, not a guess.

## Report-home (the §12 opencode side)

| opencode event | meaning | action |
|---|---|---|
| `session.idle` | turn finished → **done** | `agent_send` `--subject "opencode: done" --content "<session id>"` |
| `session.error` | errored → **BLOCKED** | `agent_send` `--subject "opencode: BLOCKED" --content "<error>"` |
| `tool.execute.after` | a tool ran → **progress** | throttled `agent_send` `--subject "opencode: progress" …` |

`agent_send` (MCP) requires `from_agent`, `to_agent`, `content`, and a `workspace`. `to_agent` =
`CORE_REPORT_TO`. `from_agent` is omitted → resolved server-side from identity, or set from
`AGENT_NAME` if present. `workspace` = `CORE_REPORT_WORKSPACE` (O1). Throttle: at most one progress
per `CORE_PROGRESS_INTERVAL_MS` per session id (module-level `Map`); idle/error never throttled.

**Silent-on-error invariant.** Hooks swallow every failure. Tools return an error *string* (never
throw) so the model sees "dispatch failed: hub unreachable" rather than the session crashing.

## File structure

```
provider/opencode/
├── package.json            # @lthn/core-agent-opencode; devDeps @opencode-ai/plugin, zod, typescript; "test": "bun test"
├── tsconfig.json           # strict, ESNext/bundler
├── src/
│   ├── plugin.ts           # entry — export const CoreAgent; wires tools + hooks
│   ├── config.ts           # loadConfig(env): pure — URL, token, target, workspace, interval
│   ├── hub.ts              # HubClient interface + makeHubClient (fetch transport) + callTool()
│   ├── tools.ts            # the five tool() factories (take HubClient)
│   ├── report.ts           # reportLifecycle() + reportProgress() (take HubClient + cfg)
│   └── throttle.ts         # shouldSend(sessionId, now): pure interval gate
├── test/
│   ├── config.test.ts      # env permutations → cfg; defaults; token-file read
│   ├── throttle.test.ts    # first passes; within-window blocked; after-window passes; per-session
│   ├── hub.test.ts         # callTool builds correct JSON-RPC body + bearer header (fake fetch); non-2xx → error result; throw → error result
│   ├── tools.test.ts       # each tool maps args → hub callTool(name,args); returns text; hub error → error string (never throws)
│   └── report.test.ts      # idle→done argv; error→BLOCKED argv; progress throttled; all swallow errors
├── AGENTS.md               # what it is + how to load (mirrors provider/codex/AGENTS.md)
└── README.md               # install + opencode.json config + env table
```

**Boundaries.** `config.ts`, `throttle.ts` are pure. `hub.ts` takes its `fetch` as a parameter
(DI) so tests assert the exact request with no network. `tools.ts`/`report.ts` take a `HubClient`
so they test against a fake. `plugin.ts` is thin opencode-facing wiring (not unit-tested; exercised
by the spike + manual load).

## Testing (`bun test`)

All units run with no network and no live hub (DI everywhere). Representative assertions:
- **config:** `loadConfig({})` → defaults (`:9202`, `cladius`, `60000`); `CORE_HUB_TOKEN_FILE` is
  read; explicit env overrides defaults.
- **throttle:** `shouldSend("s",0)===true`; `…("s",30000)===false`; `…("s",61000)===true`; per-id.
- **hub:** `callTool("agentic_status",{})` with a fake fetch → body is JSON-RPC `tools/call` with
  that name + a Bearer header; `{status:500}` → `{ok:false,error}`; fetch throws → `{ok:false}`.
- **tools:** `statusTool(fakeHub).execute({})` calls `fakeHub.callTool("agentic_status",…)` and
  returns its text; a failing hub yields an error *string*, no throw.
- **report:** `reportLifecycle(fakeHub,cfg,{type:"session.idle",…})` calls `agent_send` with
  `to_agent=cladius` + a "done" subject; `session.error` → "BLOCKED"; a throwing hub is swallowed.

No Go tests change; the Go `go build`/`go test` gate stays green (this is additive, outside `go/`).

## Build / CI

`bun install && bun test` inside `provider/opencode/`. Add a CI note (the Go gate ignores this
dir). The plugin ships as a local-dir opencode plugin and/or a published npm package; README
documents both. On the free-compute fleet, every opencode instance loads it → fleet-capable.

## Reconcile (after build) — closes part of the parity drive

- **RFC §7** — replace the `provider/opencode/` description with what ships: an `@opencode-ai/plugin`
  with the five `tool()` exports + report-home hooks, bridged to the **hub MCP plane** (note the
  `POST /mcp` attach as the documented alternative; `tool()`-export breadth + personas/skills as
  next increments). This resolves the U9 "missing provider" gap (outcome c).
- **RFC §12** — the opencode side of the report-home loop is live; update §12 (the Go-side
  push-listener half remains U10 in the parity plan).

## Open questions

- **O1 — `agent_send` workspace (OPEN).** The MCP `agent_send`/`message.send` requires a `workspace`
  (`MessageSendInput.Workspace`). In a dispatched opencode session, what is the right value — an env
  the dispatcher injects, the opencode project name, or a hub default? v1 takes it from
  `CORE_REPORT_WORKSPACE`; plan Task 1 confirms whether the dispatcher injects such an env. If there
  is no sound source, `BLOCKED.md` asks how report-home should identify its workspace (report-home
  degrades to a silent no-op until then — it never breaks the session).
- **O2 — handshake (RESOLVED).** The `:9202` REST bridge (`POST /v1/tools/<name>`) is stateless —
  no `initialize`, no `Mcp-Session-Id`. The JSON-RPC `/mcp` path (which would need the handshake)
  is the fallback only.
- **O3 — token (RESOLVED).** The bearer is the hub's `MCP_AUTH_TOKEN`; carried by the plugin's
  `CORE_HUB_TOKEN`.

## References

- opencode plugin contract — https://opencode.ai/docs/plugins/
- `go/cmd/core-agent/commands_hub.go` — the hub planes (:9201 REST, :9202 MCP)
- `external/mcp/go/pkg/mcp/transport_http.go` — `POST/GET /mcp` JSON-RPC + SSE contract
- `go/pkg/agentic/message.go` — `agent_send` / `message.send` (`from_agent`,`to_agent`,`content`,`workspace`)
- `go/pkg/agentic/dispatch.go`, `brain_client.go` — `agentic_dispatch/status/scan`, `brain_recall/remember`
- RFC §2 (hub is "the surface the plugins drive"), §7 (plugins), §12 (report-home)
- `docs/superpowers/parity/PARITY.md`, `docs/superpowers/specs/2026-05-31-rfc-parity-drive-design.md`
