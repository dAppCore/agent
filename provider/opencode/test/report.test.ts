// SPDX-License-Identifier: EUPL-1.2

import { test, expect } from "bun:test"
import { buildSendArgs, reportLifecycle, reportProgress } from "../src/report.ts"
import { Throttle } from "../src/throttle.ts"
import type { HubClient } from "../src/hub.ts"

const cfg = { reportTo: "cladius", reportWorkspace: "ws", agentName: "oc" }

function recordingHub(): { hub: HubClient; calls: Array<[string, Record<string, unknown>]> } {
  const calls: Array<[string, Record<string, unknown>]> = []
  return {
    calls,
    hub: {
      async callTool(name, args) {
        calls.push([name, args])
        return { ok: true }
      },
    },
  }
}

test("buildSendArgs: includes from_agent + workspace when set", () => {
  const a = buildSendArgs(cfg, "subj", "body")
  expect(a.to_agent).toBe("cladius")
  expect(a.from_agent).toBe("oc")
  expect(a.workspace).toBe("ws")
  expect(a.subject).toBe("subj")
  expect(a.content).toBe("body")
})

test("buildSendArgs: omits from_agent + workspace when unset", () => {
  const a = buildSendArgs({ reportTo: "cladius", reportWorkspace: null, agentName: null }, "s", "b")
  expect("from_agent" in a).toBe(false)
  expect("workspace" in a).toBe(false)
  expect(a.to_agent).toBe("cladius")
})

test("reportLifecycle: session.idle → done via agent_send", async () => {
  const { hub, calls } = recordingHub()
  await reportLifecycle(hub, cfg, { type: "session.idle", properties: { sessionID: "s1" } })
  expect(calls[0][0]).toBe("agent_send")
  expect(calls[0][1].to_agent).toBe("cladius")
  expect(String(calls[0][1].subject)).toContain("done")
})

test("reportLifecycle: session.error → BLOCKED with error detail", async () => {
  const { hub, calls } = recordingHub()
  await reportLifecycle(hub, cfg, { type: "session.error", properties: { sessionID: "s1", error: "boom" } })
  expect(String(calls[0][1].subject)).toContain("BLOCKED")
  expect(String(calls[0][1].content)).toContain("boom")
})

test("reportLifecycle: ignores unrelated events", async () => {
  const { hub, calls } = recordingHub()
  await reportLifecycle(hub, cfg, { type: "session.updated", properties: { sessionID: "s1" } })
  expect(calls.length).toBe(0)
})

test("reportLifecycle: a throwing hub is swallowed", async () => {
  const hub: HubClient = { async callTool() { throw new Error("x") } }
  // must resolve, not reject
  await reportLifecycle(hub, cfg, { type: "session.idle", properties: { sessionID: "s1" } })
  expect(true).toBe(true)
})

test("reportProgress: throttled to one per window", async () => {
  const { hub, calls } = recordingHub()
  const th = new Throttle(60000)
  await reportProgress(hub, cfg, { sessionID: "s", tool: "bash" }, th, 0)
  await reportProgress(hub, cfg, { sessionID: "s", tool: "bash" }, th, 30000)
  expect(calls.length).toBe(1)
  expect(String(calls[0][1].subject)).toContain("progress")
})

test("reportProgress: a throwing hub is swallowed", async () => {
  const hub: HubClient = { async callTool() { throw new Error("x") } }
  const th = new Throttle(0)
  await reportProgress(hub, cfg, { sessionID: "s" }, th, 0)
  expect(true).toBe(true)
})
