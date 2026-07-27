// SPDX-License-Identifier: EUPL-1.2

import { test, expect } from "bun:test"
import { loadConfig } from "../src/config.ts"

test("loadConfig: defaults when env is empty", () => {
  const c = loadConfig({})
  expect(c.hubURL).toBe("http://127.0.0.1:9202")
  expect(c.token).toBeNull()
  expect(c.reportTo).toBe("cladius")
  expect(c.reportWorkspace).toBeNull()
  expect(c.progressIntervalMs).toBe(60000)
  expect(c.agentName).toBeNull()
})

test("loadConfig: env overrides defaults", () => {
  const c = loadConfig({
    CORE_HUB_URL: "http://h:1",
    CORE_HUB_TOKEN: "tok",
    CORE_REPORT_TO: "charon",
    CORE_REPORT_WORKSPACE: "core/go-io/task-5",
    CORE_PROGRESS_INTERVAL_MS: "10",
    AGENT_NAME: "oc-1",
  })
  expect(c.hubURL).toBe("http://h:1")
  expect(c.token).toBe("tok")
  expect(c.reportTo).toBe("charon")
  expect(c.reportWorkspace).toBe("core/go-io/task-5")
  expect(c.progressIntervalMs).toBe(10)
  expect(c.agentName).toBe("oc-1")
})

test("loadConfig: blank and whitespace fall back to defaults", () => {
  const c = loadConfig({ CORE_HUB_URL: "   ", CORE_REPORT_TO: "", CORE_PROGRESS_INTERVAL_MS: "0" })
  expect(c.hubURL).toBe("http://127.0.0.1:9202")
  expect(c.reportTo).toBe("cladius")
  expect(c.progressIntervalMs).toBe(60000)
})

test("loadConfig: non-numeric interval falls back", () => {
  const c = loadConfig({ CORE_PROGRESS_INTERVAL_MS: "abc" })
  expect(c.progressIntervalMs).toBe(60000)
})
