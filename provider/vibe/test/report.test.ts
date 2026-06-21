// SPDX-License-Identifier: EUPL-1.2

import { test, expect } from "bun:test"
import { buildSendArgs, reportLifecycle, reportProgress, reportToolStart } from "../src/report.ts"
import type { HubClient } from "../src/hub.ts"
import type { Throttle } from "../src/throttle.ts"
import type { Config } from "../src/config.ts"

// Mock hub client for testing
function mockHub(
  results: Array<{ name: string; result: { ok: boolean; text?: string; error?: string } }>,
): { hub: HubClient; calls: Array<[string, Record<string, unknown>]> } {
  const calls: Array<[string, Record<string, unknown>]> = []
  let index = 0
  const hub: HubClient = {
    async callTool(name, args) {
      calls.push([name, args])
      const entry = results[index++]
      if (entry && entry.name === name) {
        return entry.result
      }
      return { ok: false, error: "not configured" }
    },
  }
  return { hub, calls }
}

// Mock throttle for testing
function mockThrottle(shouldSendFn: (sessionId: string, now: number) => boolean): Throttle {
  return {
    shouldSend: shouldSendFn,
    clear: () => {},
    clearSession: () => {},
  } as Throttle
}

const cfg: Pick<Config, "reportTo" | "reportWorkspace" | "agentName"> = {
  reportTo: "test-agent",
  reportWorkspace: "test-workspace",
  agentName: "test-vibe",
}

test("buildSendArgs: includes required fields", () => {
  const args = buildSendArgs(cfg, "test: subject", "test content")
  expect(args.to_agent).toBe("test-agent")
  expect(args.subject).toBe("test: subject")
  expect(args.content).toBe("test content")
  expect(args.from_agent).toBe("test-vibe")
  expect(args.workspace).toBe("test-workspace")
})

test("buildSendArgs: omits optional fields when null", () => {
  const partialCfg = { reportTo: "test-agent", reportWorkspace: null, agentName: null } as const
  const args = buildSendArgs(partialCfg, "subject", "content")
  expect(args.to_agent).toBe("test-agent")
  expect(args.from_agent).toBeUndefined()
  expect(args.workspace).toBeUndefined()
})

test("reportLifecycle: reports session.end → done", async () => {
  const { hub, calls } = mockHub([
    { name: "agent_send", result: { ok: true } },
  ])
  await reportLifecycle(hub, cfg, {
    type: "session.end",
    properties: { sessionID: "s1", agent: "vibe" },
  })
  expect(calls.length).toBe(1)
  expect(calls[0][0]).toBe("agent_send")
  const args = calls[0][1] as Record<string, unknown>
  expect(args.subject).toBe("vibe: done")
  expect(args.content).toContain("session s1")
})

test("reportLifecycle: reports session.error → BLOCKED", async () => {
  const { hub, calls } = mockHub([
    { name: "agent_send", result: { ok: true } },
  ])
  await reportLifecycle(hub, cfg, {
    type: "session.error",
    properties: { sessionID: "s1", error: "test error" },
  })
  expect(calls.length).toBe(1)
  const args = calls[0][1] as Record<string, unknown>
  expect(args.subject).toBe("vibe: BLOCKED")
  expect(args.content).toContain("test error")
})

test("reportLifecycle: reports message.completed", async () => {
  const { hub, calls } = mockHub([
    { name: "agent_send", result: { ok: true } },
  ])
  await reportLifecycle(hub, cfg, {
    type: "message.completed",
    properties: { sessionID: "s1" },
  })
  expect(calls.length).toBe(1)
  const args = calls[0][1] as Record<string, unknown>
  expect(args.subject).toBe("vibe: message")
})

test("reportLifecycle: ignores unknown event types", async () => {
  const { hub, calls } = mockHub([])
  await reportLifecycle(hub, cfg, {
    type: "unknown.event",
    properties: { sessionID: "s1" },
  })
  expect(calls.length).toBe(0)
})

test("reportLifecycle: never throws on hub error", async () => {
  const { hub } = mockHub([
    { name: "agent_send", result: { ok: false, error: "hub error" } },
  ])
  await expect(
    reportLifecycle(hub, cfg, { type: "session.end", properties: { sessionID: "s1" } }),
  ).resolves.toBeUndefined()
})

test("reportProgress: sends when throttle allows", async () => {
  const { hub, calls } = mockHub([
    { name: "agent_send", result: { ok: true } },
  ])
  const throttle = mockThrottle(() => true)
  await reportProgress(hub, cfg, { sessionID: "s1", tool: "test_tool", agent: "vibe" }, throttle, 1000)
  expect(calls.length).toBe(1)
  const args = calls[0][1] as Record<string, unknown>
  expect(args.subject).toBe("vibe: progress")
  expect(args.content).toContain("test_tool")
})

test("reportProgress: skips when throttle blocks", async () => {
  const { hub, calls } = mockHub([])
  const throttle = mockThrottle(() => false)
  await reportProgress(hub, cfg, { sessionID: "s1", tool: "test_tool", agent: "vibe" }, throttle, 1000)
  expect(calls.length).toBe(0)
})

test("reportProgress: never throws on hub error", async () => {
  const { hub } = mockHub([
    { name: "agent_send", result: { ok: false, error: "hub error" } },
  ])
  const throttle = mockThrottle(() => true)
  await expect(
    reportProgress(hub, cfg, { sessionID: "s1", tool: "test_tool", agent: "vibe" }, throttle, 1000),
  ).resolves.toBeUndefined()
})

test("reportToolStart: reports tool start", async () => {
  const { hub, calls } = mockHub([
    { name: "agent_send", result: { ok: true } },
  ])
  await reportToolStart(hub, cfg, { sessionID: "s1", tool: "test_tool", args: { key: "value" } })
  expect(calls.length).toBe(1)
  const args = calls[0][1] as Record<string, unknown>
  expect(args.subject).toBe("vibe: tool_start")
  expect(args.content).toContain("test_tool")
  expect(args.content).toContain("key")
})

test("reportToolStart: never throws on hub error", async () => {
  const { hub } = mockHub([
    { name: "agent_send", result: { ok: false, error: "hub error" } },
  ])
  await expect(
    reportToolStart(hub, cfg, { sessionID: "s1", tool: "test_tool" }),
  ).resolves.toBeUndefined()
})

test("reportLifecycle: uses default agent name", async () => {
  const { hub, calls } = mockHub([
    { name: "agent_send", result: { ok: true } },
  ])
  const partialCfg = { reportTo: "test-agent", reportWorkspace: null, agentName: null } as const
  await reportLifecycle(hub, partialCfg, {
    type: "session.end",
    properties: { sessionID: "s1" },
  })
  const args = calls[0][1] as Record<string, unknown>
  expect(args.content).toContain("(vibe)")
})
