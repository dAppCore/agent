<!-- SPDX-Licence-Identifier: EUPL-1.2 -->

# `provider/opencode` Plugin Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL — use `superpowers:subagent-driven-development` or
> `superpowers:executing-plans`. Steps use checkbox (`- [ ]`) syntax. TDD throughout: failing test
> → minimal code → green → commit.

**Goal:** Ship `provider/opencode/` — an `@opencode-ai/plugin` that exposes core-agent's
`dispatch/status/scan/brain_recall/brain_remember` as `tool()`s bridged to the hub MCP plane
(:9202), and reports session lifecycle home via `agent_send`, never breaking the session.

**Architecture:** TypeScript, tested with `bun test`. Pure modules (`config`, `throttle`) +
DI-wrapped transport (`hub` takes `fetch`; `tools`/`report` take a `HubClient`) so every unit tests
with no network. Thin `plugin.ts` wires opencode events/tools to the modules.

**Tech Stack:** Bun 1.3 (runtime + test), `@opencode-ai/plugin`, `zod` (via `tool.schema`),
TypeScript strict. Bridges to `core-agent hub` over HTTP JSON-RPC 2.0.

---

### Task 1: Spike — confirm arg keys + O1 (transport already resolved)

**Goal:** Transport is settled by code-read (O2/O3 resolved — see spec): the v1 transport is the
stateless REST bridge `POST {base}/v1/tools/<tool_name>`, Bearer = `MCP_AUTH_TOKEN`, body = args
object, result JSON. This task only confirms each tool's **exact arg keys** and resolves **O1**
(the `agent_send` workspace value). Investigation, not TDD.

- [ ] **Step 1:** Start a hub: `cd go && MCP_AUTH_TOKEN=devtok MCP_JWT_SECRET=devsecret go run
  ./cmd/core-agent hub --mcp-http 127.0.0.1:9202 --no-http` (or reuse a running one).
- [ ] **Step 2:** Hit the bridge to confirm shape + arg keys (no JSON-RPC):
  `curl -s -X POST localhost:9202/v1/tools/agentic_status -H 'Authorization: Bearer devtok'
  -H 'Content-Type: application/json' -d '{}'` — repeat for `agentic_dispatch`, `agentic_scan`,
  `brain_recall`, `brain_remember`, `agent_send`; record the arg keys each accepts/requires.
  (If a bare bridge call needs no extra handshake — expected — O2 is confirmed empirically too.)
- [ ] **Step 3 — O1:** Determine `agent_send`'s `workspace` source: grep how dispatch injects env
  into the opencode container (`go/pkg/agentic/container.go`, `dispatch.go`) for a workspace/agent
  identity var the plugin can read. Record the answer (env name) or that none exists.
- [ ] **Step 4:** Update the spec's "Open questions" (O1 resolved or escalated) and the tool arg
  tables if the spike found different keys.
- [ ] **Step 5:** If O1 has no sound source AND report-home is required for v1 acceptance →
  `BLOCKED.md`. Otherwise proceed: report-home degrades to a silent no-op when `CORE_REPORT_WORKSPACE`
  is unset (never breaks the session), which is an acceptable v1 state.

> If a live hub cannot be started here, build Tasks 2–9 against the confirmed REST-bridge shape (the
> modules are DI'd, so they're correct regardless) and mark Step 2/3 as a follow-up to run before
> first real use. Note this in the README.

### Task 2: Scaffold

**Files:** Create `provider/opencode/package.json`, `provider/opencode/tsconfig.json`,
`provider/opencode/.gitignore`.

- [ ] **Step 1:** `package.json` — name `@lthn/core-agent-opencode`, `"type":"module"`,
  `"test":"bun test"`, devDeps `@opencode-ai/plugin`, `typescript`; license `EUPL-1.2`.
- [ ] **Step 2:** `tsconfig.json` — `strict`, `module:"ESNext"`, `moduleResolution:"bundler"`,
  `types:["bun-types"]`.
- [ ] **Step 3:** `.gitignore` — `node_modules`, `*.tsbuildinfo`.
- [ ] **Step 4:** `bun install` → lockfile resolves. **Commit** `chore(opencode): scaffold plugin`.

### Task 3: `config.ts` (pure) — TDD

**Files:** Create `src/config.ts`, `test/config.test.ts`.

- [ ] **Step 1 — failing test** (`test/config.test.ts`):
```typescript
import { test, expect } from "bun:test"
import { loadConfig } from "../src/config"

test("defaults", () => {
  const c = loadConfig({})
  expect(c.hubURL).toBe("http://127.0.0.1:9202")
  expect(c.reportTo).toBe("cladius")
  expect(c.progressIntervalMs).toBe(60000)
  expect(c.token).toBeNull()
})
test("env overrides", () => {
  const c = loadConfig({ CORE_HUB_URL: "http://h:1", CORE_HUB_TOKEN: "t", CORE_REPORT_TO: "x", CORE_PROGRESS_INTERVAL_MS: "10" })
  expect(c.hubURL).toBe("http://h:1"); expect(c.token).toBe("t"); expect(c.reportTo).toBe("x"); expect(c.progressIntervalMs).toBe(10)
})
```
- [ ] **Step 2:** Run `bun test test/config.test.ts` → FAIL (no module).
- [ ] **Step 3 — implement** `src/config.ts`:
```typescript
// SPDX-License-Identifier: EUPL-1.2
export interface Config {
  hubURL: string; token: string | null; reportTo: string
  reportWorkspace: string | null; progressIntervalMs: number; agentName: string | null
}
export function loadConfig(env: Record<string, string | undefined>): Config {
  const tokenFromFile = env.CORE_HUB_TOKEN_FILE ? readFileSafe(env.CORE_HUB_TOKEN_FILE) : null
  return {
    hubURL: env.CORE_HUB_URL?.trim() || "http://127.0.0.1:9202",
    token: (env.CORE_HUB_TOKEN?.trim() || tokenFromFile) ?? null,
    reportTo: env.CORE_REPORT_TO?.trim() || "cladius",
    reportWorkspace: env.CORE_REPORT_WORKSPACE?.trim() || null,
    progressIntervalMs: Number(env.CORE_PROGRESS_INTERVAL_MS) || 60000,
    agentName: env.AGENT_NAME?.trim() || null,
  }
}
function readFileSafe(p: string): string | null {
  try { return require("node:fs").readFileSync(p, "utf8").trim() || null } catch { return null }
}
```
- [ ] **Step 4:** Run → PASS. **Step 5: Commit** `feat(opencode): config loader`.

### Task 4: `throttle.ts` (pure) — TDD

**Files:** Create `src/throttle.ts`, `test/throttle.test.ts`.

- [ ] **Step 1 — failing test:**
```typescript
import { test, expect } from "bun:test"
import { Throttle } from "../src/throttle"
test("interval gate per session", () => {
  const t = new Throttle(60000)
  expect(t.shouldSend("s", 0)).toBe(true)
  expect(t.shouldSend("s", 30000)).toBe(false)
  expect(t.shouldSend("s", 61000)).toBe(true)
  expect(t.shouldSend("other", 30000)).toBe(true)
})
```
- [ ] **Step 2:** Run → FAIL.
- [ ] **Step 3 — implement:**
```typescript
// SPDX-License-Identifier: EUPL-1.2
export class Throttle {
  private last = new Map<string, number>()
  constructor(private intervalMs: number) {}
  shouldSend(sessionId: string, now: number): boolean {
    const prev = this.last.get(sessionId)
    if (prev !== undefined && now - prev < this.intervalMs) return false
    this.last.set(sessionId, now); return true
  }
}
```
- [ ] **Step 4:** Run → PASS. **Step 5: Commit** `feat(opencode): progress throttle`.

### Task 5: `hub.ts` (DI transport) — TDD

**Files:** Create `src/hub.ts`, `test/hub.test.ts`.

- [ ] **Step 1 — failing test** (inject a fake `fetch`):
```typescript
import { test, expect } from "bun:test"
import { makeHubClient } from "../src/hub"
test("callTool builds JSON-RPC + bearer", async () => {
  let seen: any
  const fakeFetch = async (url: string, init: any) => {
    seen = { url, init }
    return new Response(JSON.stringify({ jsonrpc: "2.0", id: 1, result: { content: [{ type: "text", text: "ok" }] } }), { status: 200 })
  }
  const hub = makeHubClient({ hubURL: "http://h:9202", token: "t" } as any, fakeFetch as any)
  const r = await hub.callTool("agentic_status", { workspace: "w" })
  expect(r.ok).toBe(true); expect(r.text).toBe("ok")
  expect(seen.url).toBe("http://h:9202/mcp")
  expect(seen.init.headers.Authorization).toBe("Bearer t")
  const body = JSON.parse(seen.init.body)
  expect(body.method).toBe("tools/call"); expect(body.params.name).toBe("agentic_status")
  expect(body.params.arguments).toEqual({ workspace: "w" })
})
test("non-2xx → error result, never throws", async () => {
  const hub = makeHubClient({ hubURL: "http://h", token: "t" } as any, (async () => new Response("nope", { status: 500 })) as any)
  const r = await hub.callTool("x", {}); expect(r.ok).toBe(false)
})
test("fetch throws → error result", async () => {
  const hub = makeHubClient({ hubURL: "http://h", token: "t" } as any, (async () => { throw new Error("down") }) as any)
  const r = await hub.callTool("x", {}); expect(r.ok).toBe(false)
})
test("no token → error result, no fetch", async () => {
  let called = false
  const hub = makeHubClient({ hubURL: "http://h", token: null } as any, (async () => { called = true; return new Response("") }) as any)
  const r = await hub.callTool("x", {}); expect(r.ok).toBe(false); expect(called).toBe(false)
})
```
- [ ] **Step 2:** Run → FAIL.
- [ ] **Step 3 — implement** `src/hub.ts` against the **REST bridge**: `callTool(name,args)` →
  `POST {hubURL}/v1/tools/{name}` with `Authorization: Bearer {token}`,
  `Content-Type: application/json`, body `JSON.stringify(args)`. Result text = the JSON response
  body stringified (or its `.text`/`.content[].text` if present). No token → `{ok:false}` without
  fetching; non-2xx or throw → `{ok:false,error}`. Signature:
  `export interface HubClient { callTool(name: string, args: Record<string, unknown>): Promise<{ok:boolean; text?:string; error?:string}> }`
  and `export function makeHubClient(cfg, fetchImpl = fetch): HubClient`.
- [ ] **Step 4:** Run → PASS. **Step 5: Commit** `feat(opencode): hub REST-bridge client (DI fetch)`.

> The test in Step 1 above asserts `seen.url === "http://h:9202/v1/tools/agentic_status"` and the
> body equals the args object directly (no JSON-RPC envelope). Update the Step-1 test's URL/body
> expectations to the REST-bridge shape before implementing. The JSON-RPC `/mcp` path stays a
> fallback behind the same interface if ever needed.

### Task 6: `tools.ts` (DI on HubClient) — TDD

**Files:** Create `src/tools.ts`, `test/tools.test.ts`.

- [ ] **Step 1 — failing test** (fake HubClient; assert mapping + never-throws):
```typescript
import { test, expect } from "bun:test"
import { buildTools } from "../src/tools"
const fakeHub = (rec: any[]) => ({ callTool: async (n: string, a: any) => { rec.push([n, a]); return { ok: true, text: "done" } } })
test("status maps to agentic_status", async () => {
  const rec: any[] = []; const tools = buildTools(fakeHub(rec) as any)
  const out = await tools.status.execute({ workspace: "w" }, {} as any)
  expect(rec[0][0]).toBe("agentic_status"); expect(out).toContain("done")
})
test("dispatch maps to agentic_dispatch", async () => {
  const rec: any[] = []; const tools = buildTools(fakeHub(rec) as any)
  await tools.dispatch.execute({ repo: "r", task: "t" }, {} as any)
  expect(rec[0][0]).toBe("agentic_dispatch"); expect(rec[0][1].repo).toBe("r")
})
test("hub error → error string, never throws", async () => {
  const hub = { callTool: async () => ({ ok: false, error: "hub down" }) }
  const tools = buildTools(hub as any)
  const out = await tools.brain_recall.execute({ query: "q" }, {} as any)
  expect(out).toContain("hub down")
})
```
- [ ] **Step 2:** Run → FAIL.
- [ ] **Step 3 — implement** `src/tools.ts`: `buildTools(hub: HubClient)` returns
  `{ dispatch, status, scan, brain_recall, brain_remember }`, each via `tool({description, args:
  {…tool.schema}, execute})`. `execute` calls `hub.callTool(<mcpName>, args)` and returns
  `r.ok ? r.text! : "<tool> failed: " + r.error`. Arg keys per Task 1 (default to the spec table).
- [ ] **Step 4:** Run → PASS. **Step 5: Commit** `feat(opencode): five tool() exports`.

### Task 7: `report.ts` (DI on HubClient) — TDD

**Files:** Create `src/report.ts`, `test/report.test.ts`.

- [ ] **Step 1 — failing test:**
```typescript
import { test, expect } from "bun:test"
import { reportLifecycle, reportProgress } from "../src/report"
import { Throttle } from "../src/throttle"
const cfg = { reportTo: "cladius", reportWorkspace: "ws", agentName: "oc" } as any
test("idle → done via agent_send", async () => {
  const rec: any[] = []
  const hub = { callTool: async (n: string, a: any) => { rec.push([n, a]); return { ok: true, text: "" } } }
  await reportLifecycle(hub as any, cfg, { type: "session.idle", properties: { sessionID: "s" } })
  expect(rec[0][0]).toBe("agent_send"); expect(rec[0][1].to_agent).toBe("cladius")
  expect(String(rec[0][1].subject)).toContain("done")
})
test("error → BLOCKED", async () => {
  const rec: any[] = []
  const hub = { callTool: async (n: string, a: any) => { rec.push([n, a]); return { ok: true } } }
  await reportLifecycle(hub as any, cfg, { type: "session.error", properties: { sessionID: "s", error: "boom" } })
  expect(String(rec[0][1].subject)).toContain("BLOCKED")
})
test("throwing hub is swallowed", async () => {
  const hub = { callTool: async () => { throw new Error("x") } }
  await reportLifecycle(hub as any, cfg, { type: "session.idle", properties: { sessionID: "s" } }) // must not throw
})
test("progress throttled", async () => {
  const rec: any[] = []
  const hub = { callTool: async (n: string, a: any) => { rec.push(n); return { ok: true } } }
  const th = new Throttle(60000)
  await reportProgress(hub as any, cfg, { sessionID: "s" }, th, 0)
  await reportProgress(hub as any, cfg, { sessionID: "s" }, th, 30000)
  expect(rec.length).toBe(1)
})
```
- [ ] **Step 2:** Run → FAIL.
- [ ] **Step 3 — implement** `src/report.ts`: `reportLifecycle(hub,cfg,event)` switches on
  `event.type` (`session.idle`→done, `session.error`→BLOCKED), builds the `agent_send` args
  (`from_agent: cfg.agentName ?? undefined, to_agent: cfg.reportTo, workspace: cfg.reportWorkspace,
  subject, content`), and is wrapped `try{…}catch{}`. `reportProgress(hub,cfg,input,throttle,now)`
  gates on `throttle.shouldSend(input.sessionID, now)` then sends; also try/catch-swallowed.
- [ ] **Step 4:** Run → PASS. **Step 5: Commit** `feat(opencode): report-home hooks`.

### Task 8: `plugin.ts` (wiring) + full suite

**Files:** Create `src/plugin.ts`.

- [ ] **Step 1:** Implement the entry (matches the spec's "What it is" block): `loadConfig(process.env)`
  → `makeHubClient(cfg)` → `buildTools(hub)` → return `{ tool: {...}, event, "tool.execute.after" }`.
  `event` calls `reportLifecycle`; `tool.execute.after` calls `reportProgress` with a module-level
  `Throttle(cfg.progressIntervalMs)` and `Date.now()`.
- [ ] **Step 2:** Run the **whole** suite: `cd provider/opencode && bun test` → all PASS.
- [ ] **Step 3:** `bunx tsc --noEmit` → no type errors. **Step 4: Commit** `feat(opencode): plugin entry + wiring`.

### Task 9: Docs

**Files:** Create `provider/opencode/AGENTS.md`, `provider/opencode/README.md`.

- [ ] **Step 1:** `AGENTS.md` — what the plugin is, the five tools, the report-home behaviour
  (mirror `provider/codex/AGENTS.md` tone).
- [ ] **Step 2:** `README.md` — install (`opencode.json` `"plugin"` entry + local-dir), the env
  table (`CORE_HUB_URL/TOKEN/TOKEN_FILE/REPORT_TO/REPORT_WORKSPACE/PROGRESS_INTERVAL_MS`), and a
  note on Task 1 (run the spike before first real use if it was deferred).
- [ ] **Step 3: Commit** `docs(opencode): AGENTS + README`.

### Task 10: Reconcile RFC (closes U9 / part of §12)

**Files:** Modify `RFC.md` §7 and §12; update `docs/superpowers/parity/PARITY.md`.

- [ ] **Step 1:** RFC §7 — rewrite the `provider/opencode/` bullet to describe what shipped (five
  `tool()` exports + report-home hooks over the hub MCP plane; note `POST /mcp` attach + breadth/
  personas/skills as next increments).
- [ ] **Step 2:** RFC §12 — note the opencode side of the report-home loop is live (Go-side
  push-listener remains U10).
- [ ] **Step 3:** `PARITY.md` — mark §7 `provider/opencode` resolved (outcome c, built).
- [ ] **Step 4:** Gate stays green: `cd go && go build ./... && go test ./... -count=1 -timeout 120s`
  (unchanged — additive). **Step 5: Commit** `docs(agent): reconcile RFC §7/§12 — opencode plugin shipped`.

## Self-review

- **Spec coverage:** transport (Task 5), five tools (Task 6), report-home (Task 7), config/throttle
  (Tasks 3/4), wiring (Task 8), docs (Task 9), reconcile (Task 10), open questions (Task 1). ✓
- **No placeholders:** every code step shows real code or a precise signature + the test that pins
  it; the only deferrals (exact arg keys, MCP handshake) are explicitly routed through Task 1's
  spike, not hand-waved. ✓
- **Type consistency:** `HubClient.callTool(name,args)→{ok,text?,error?}` used identically in Tasks
  5/6/7; `loadConfig→Config` fields match `report.ts`/`plugin.ts` usage; `Throttle.shouldSend`
  signature consistent Tasks 4/7. ✓
