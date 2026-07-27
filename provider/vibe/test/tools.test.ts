// SPDX-License-Identifier: EUPL-1.2

import { test, expect } from "bun:test"
import { TOOL_MAP, runTool, runToolDynamic, getToolList, getToolDescription } from "../src/tool_exec.ts"
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

test("TOOL_MAP: Vibe names map to the right hub tools", () => {
  expect(TOOL_MAP.dispatch).toBe("agentic_dispatch")
  expect(TOOL_MAP.status).toBe("agentic_status")
  expect(TOOL_MAP.scan).toBe("agentic_scan")
  expect(TOOL_MAP.brain_recall).toBe("brain_recall")
  expect(TOOL_MAP.brain_remember).toBe("brain_remember")
  expect(TOOL_MAP.agent_send).toBe("agent_send")
  expect(TOOL_MAP.agent_inbox).toBe("agent_inbox")
  expect(TOOL_MAP.agent_conversation).toBe("agent_conversation")
})

test("TOOL_MAP: includes all tool categories", () => {
  // Dispatch
  expect(TOOL_MAP.dispatch).toBeDefined()
  expect(TOOL_MAP.dispatch_remote).toBeDefined()
  expect(TOOL_MAP.status).toBeDefined()
  expect(TOOL_MAP.status_remote).toBeDefined()

  // Workspace
  expect(TOOL_MAP.prep_workspace).toBeDefined()
  expect(TOOL_MAP.resume).toBeDefined()
  expect(TOOL_MAP.watch).toBeDefined()

  // PR/Review
  expect(TOOL_MAP.create_pr).toBeDefined()
  expect(TOOL_MAP.list_prs).toBeDefined()
  expect(TOOL_MAP.create_epic).toBeDefined()
  expect(TOOL_MAP.review_queue).toBeDefined()

  // Mirror
  expect(TOOL_MAP.mirror).toBeDefined()

  // Scan
  expect(TOOL_MAP.scan).toBeDefined()

  // Brain
  expect(TOOL_MAP.brain_recall).toBeDefined()
  expect(TOOL_MAP.brain_remember).toBeDefined()
  expect(TOOL_MAP.brain_forget).toBeDefined()

  // Messaging
  expect(TOOL_MAP.agent_send).toBeDefined()
  expect(TOOL_MAP.agent_inbox).toBeDefined()
  expect(TOOL_MAP.agent_conversation).toBeDefined()

  // Plans
  expect(TOOL_MAP.plan_create).toBeDefined()
  expect(TOOL_MAP.plan_read).toBeDefined()
  expect(TOOL_MAP.plan_update).toBeDefined()
  expect(TOOL_MAP.plan_delete).toBeDefined()
  expect(TOOL_MAP.plan_list).toBeDefined()

  // Files
  expect(TOOL_MAP.file_read).toBeDefined()
  expect(TOOL_MAP.file_write).toBeDefined()
  expect(TOOL_MAP.file_edit).toBeDefined()
  expect(TOOL_MAP.file_delete).toBeDefined()
  expect(TOOL_MAP.file_rename).toBeDefined()
  expect(TOOL_MAP.file_exists).toBeDefined()
  expect(TOOL_MAP.dir_list).toBeDefined()
  expect(TOOL_MAP.dir_create).toBeDefined()

  // Language
  expect(TOOL_MAP.lang_detect).toBeDefined()
  expect(TOOL_MAP.lang_list).toBeDefined()
})

test("getToolList: returns all tool names", () => {
  const tools = getToolList()
  expect(tools).toContain("dispatch")
  expect(tools).toContain("status")
  expect(tools).toContain("scan")
  expect(tools.length).toBeGreaterThan(20)
})

test("getToolDescription: returns descriptions for known tools", () => {
  expect(getToolDescription("dispatch")).toContain("Dispatch")
  expect(getToolDescription("status")).toContain("status")
  expect(getToolDescription("brain_recall")).toContain("OpenBrain")
})

test("getToolDescription: returns fallback for unknown tool", () => {
  const desc = getToolDescription("unknown_tool" as any)
  expect(desc).toContain("core-agent tool")
})

test("runTool: forwards name + args and returns hub text", async () => {
  const { hub, calls } = recordingHub({ ok: true, text: "dispatched" })
  const out = await runTool(hub, "dispatch", { repo: "r", task: "t" })
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

test("runToolDynamic: calls tool by name", async () => {
  const { hub, calls } = recordingHub({ ok: true, text: "dynamic result" })
  const out = await runToolDynamic(hub, "custom_tool", { arg: "value" })
  expect(out).toBe("dynamic result")
  expect(calls[0][0]).toBe("custom_tool")
  expect(calls[0][1]).toEqual({ arg: "value" })
})

test("runToolDynamic: hub failure → error string", async () => {
  const hub: HubClient = { async callTool() { return { ok: false, error: "not found" } } }
  const out = await runToolDynamic(hub, "unknown_tool", {})
  expect(out).toContain("not found")
  expect(out).toContain("unknown_tool failed")
})
