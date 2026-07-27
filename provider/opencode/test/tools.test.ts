// SPDX-License-Identifier: EUPL-1.2

import { test, expect } from "bun:test"
import { TOOL_MAP, runTool } from "../src/tool_exec.ts"
import type { HubClient } from "../src/hub.ts"

function recordingHub(result = { ok: true, text: "done" }): { hub: HubClient; calls: Array<[string, Record<string, unknown>]> } {
  const calls: Array<[string, Record<string, unknown>]> = []
  const hub: HubClient = {
    async callTool(name, args) {
      calls.push([name, args])
      return result
    },
  }
  return { hub, calls }
}

test("TOOL_MAP: opencode names map to the right hub tools", () => {
  expect(TOOL_MAP.dispatch).toBe("agentic_dispatch")
  expect(TOOL_MAP.status).toBe("agentic_status")
  expect(TOOL_MAP.scan).toBe("agentic_scan")
  expect(TOOL_MAP.brain_recall).toBe("brain_recall")
  expect(TOOL_MAP.brain_remember).toBe("brain_remember")
})

test("runTool: forwards name + args and returns hub text", async () => {
  const { hub, calls } = recordingHub({ ok: true, text: "dispatched" })
  const out = await runTool(hub, TOOL_MAP.dispatch, { repo: "r", task: "t" })
  expect(out).toBe("dispatched")
  expect(calls[0][0]).toBe("agentic_dispatch")
  expect(calls[0][1]).toEqual({ repo: "r", task: "t" })
})

test("runTool: hub failure → error string, never throws", async () => {
  const hub: HubClient = { async callTool() { return { ok: false, error: "hub down" } } }
  const out = await runTool(hub, "brain_recall", { query: "q" })
  expect(out).toContain("hub down")
  expect(out).toContain("brain_recall failed")
})

test("runTool: ok with no text → empty string", async () => {
  const hub: HubClient = { async callTool() { return { ok: true } } }
  const out = await runTool(hub, "agentic_status", {})
  expect(out).toBe("")
})
