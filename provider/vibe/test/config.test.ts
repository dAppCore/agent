// SPDX-License-Identifier: EUPL-1.2

import { test, expect } from "bun:test"
import { loadConfig, isToolEnabled } from "../src/config.ts"

test("loadConfig: defaults with empty env", () => {
  const cfg = loadConfig({})
  expect(cfg.hubURL).toBe("http://127.0.0.1:9202")
  expect(cfg.token).toBeNull()
  expect(cfg.reportTo).toBe("cladius")
  expect(cfg.reportWorkspace).toBeNull()
  expect(cfg.progressIntervalMs).toBe(60000)
  expect(cfg.agentName).toBeNull()
  expect(cfg.enabledTools).toEqual([])
})

test("loadConfig: reads hub URL from env", () => {
  const cfg = loadConfig({ CORE_HUB_URL: "http://custom:8080" })
  expect(cfg.hubURL).toBe("http://custom:8080")
})

test("loadConfig: reads token from env", () => {
  const cfg = loadConfig({ CORE_HUB_TOKEN: "secret-token" })
  expect(cfg.token).toBe("secret-token")
})

test("loadConfig: reads reportTo from env", () => {
  const cfg = loadConfig({ CORE_REPORT_TO: "agent-x" })
  expect(cfg.reportTo).toBe("agent-x")
})

test("loadConfig: reads reportWorkspace from env", () => {
  const cfg = loadConfig({ CORE_REPORT_WORKSPACE: "workspace-y" })
  expect(cfg.reportWorkspace).toBe("workspace-y")
})

test("loadConfig: reads progressIntervalMs from env", () => {
  const cfg = loadConfig({ CORE_PROGRESS_INTERVAL_MS: "30000" })
  expect(cfg.progressIntervalMs).toBe(30000)
})

test("loadConfig: reads agentName from env", () => {
  const cfg = loadConfig({ AGENT_NAME: "test-agent" })
  expect(cfg.agentName).toBe("test-agent")
})

test("loadConfig: reads enabledTools from env", () => {
  const cfg = loadConfig({ CORE_VIBE_ENABLED_TOOLS: "dispatch,status,scan" })
  expect(cfg.enabledTools).toEqual(["dispatch", "status", "scan"])
})

test("loadConfig: trims whitespace", () => {
  const cfg = loadConfig({ CORE_HUB_URL: "  http://test:8080  " })
  expect(cfg.hubURL).toBe("http://test:8080")
})

test("loadConfig: handles empty string values as unset", () => {
  const cfg = loadConfig({ CORE_HUB_TOKEN: "" })
  expect(cfg.token).toBeNull()
})

test("isToolEnabled: returns true when enabledTools is empty", () => {
  const cfg = { enabledTools: [] } as const
  expect(isToolEnabled("dispatch", cfg)).toBe(true)
  expect(isToolEnabled("status", cfg)).toBe(true)
})

test("isToolEnabled: returns true for enabled tool", () => {
  const cfg = { enabledTools: ["dispatch", "status"] } as const
  expect(isToolEnabled("dispatch", cfg)).toBe(true)
  expect(isToolEnabled("status", cfg)).toBe(true)
  expect(isToolEnabled("scan", cfg)).toBe(false)
})

test("isToolEnabled: returns false for disabled tool", () => {
  const cfg = { enabledTools: ["dispatch"] } as const
  expect(isToolEnabled("status", cfg)).toBe(false)
})
