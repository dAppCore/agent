// SPDX-License-Identifier: EUPL-1.2

import { test, expect } from "bun:test"
import { makeHubClient, type HubClient, type HubResult } from "../src/hub.ts"
import type { Config } from "../src/config.ts"

// Mock fetch implementation for testing
function mockFetch(responses: Map<string, { status: number; body: string }>): (
  url: string,
  init: { method: string; headers: Record<string, string>; body: string },
) => Promise<Response> {
  return async (url, init) => {
    const key = `${init.method}:${url}:${init.body}`
    const response = responses.get(url) ?? { status: 500, body: "Not found" }
    return new Response(response.body, {
      status: response.status,
      headers: { "Content-Type": "application/json" },
    })
  }
}

test("makeHubClient: no token returns error", async () => {
  const cfg = { hubURL: "http://test:8080", token: null } as Pick<Config, "hubURL" | "token">
  const hub = makeHubClient(cfg)
  const r = await hub.callTool("test_tool", {})
  expect(r.ok).toBe(false)
  expect(r.error).toContain("token not configured")
})

test("makeHubClient: success call with token", async () => {
  const cfg = { hubURL: "http://test:8080", token: "test-token" } as Pick<Config, "hubURL" | "token">
  const responses = new Map<string, { status: number; body: string }>()
  responses.set("http://test:8080/v1/tools/test_tool", {
    status: 200,
    body: JSON.stringify({ text: "success" }),
  })
  const hub = makeHubClient(cfg, mockFetch(responses))
  const r = await hub.callTool("test_tool", {})
  expect(r.ok).toBe(true)
  expect(r.text).toBe("success")
})

test("makeHubClient: strips trailing slash from hubURL", async () => {
  const cfg = { hubURL: "http://test:8080/", token: "test-token" } as Pick<Config, "hubURL" | "token">
  const responses = new Map<string, { status: number; body: string }>()
  responses.set("http://test:8080/v1/tools/test_tool", {
    status: 200,
    body: JSON.stringify({ text: "success" }),
  })
  const hub = makeHubClient(cfg, mockFetch(responses))
  const r = await hub.callTool("test_tool", {})
  expect(r.ok).toBe(true)
  expect(r.text).toBe("success")
})

test("makeHubClient: extracts text from response", async () => {
  const cfg = { hubURL: "http://test:8080", token: "test-token" } as Pick<Config, "hubURL" | "token">
  const responses = new Map<string, { status: number; body: string }>()
  responses.set("http://test:8080/v1/tools/test_tool", {
    status: 200,
    body: JSON.stringify({ text: "extracted text" }),
  })
  const hub = makeHubClient(cfg, mockFetch(responses))
  const r = await hub.callTool("test_tool", {})
  expect(r.text).toBe("extracted text")
})

test("makeHubClient: extracts content array text from response", async () => {
  const cfg = { hubURL: "http://test:8080", token: "test-token" } as Pick<Config, "hubURL" | "token">
  const responses = new Map<string, { status: number; body: string }>()
  responses.set("http://test:8080/v1/tools/test_tool", {
    status: 200,
    body: JSON.stringify({ content: [{ text: "part1" }, { text: "part2" }] }),
  })
  const hub = makeHubClient(cfg, mockFetch(responses))
  const r = await hub.callTool("test_tool", {})
  expect(r.text).toBe("part1\npart2")
})

test("makeHubClient: returns raw body when not JSON", async () => {
  const cfg = { hubURL: "http://test:8080", token: "test-token" } as Pick<Config, "hubURL" | "token">
  const responses = new Map<string, { status: number; body: string }>()
  responses.set("http://test:8080/v1/tools/test_tool", {
    status: 200,
    body: "raw text response",
  })
  const hub = makeHubClient(cfg, mockFetch(responses))
  const r = await hub.callTool("test_tool", {})
  expect(r.text).toBe("raw text response")
})

test("makeHubClient: handles non-2xx status", async () => {
  const cfg = { hubURL: "http://test:8080", token: "test-token" } as Pick<Config, "hubURL" | "token">
  const responses = new Map<string, { status: number; body: string }>()
  responses.set("http://test:8080/v1/tools/test_tool", {
    status: 404,
    body: "Not found",
  })
  const hub = makeHubClient(cfg, mockFetch(responses))
  const r = await hub.callTool("test_tool", {})
  expect(r.ok).toBe(false)
  expect(r.error).toContain("404")
})

test("makeHubClient: handles network error", async () => {
  const cfg = { hubURL: "http://test:8080", token: "test-token" } as Pick<Config, "hubURL" | "token">
  const failingFetch: typeof fetch = async () => {
    throw new Error("Network error")
  }
  const hub = makeHubClient(cfg, failingFetch as unknown as typeof fetch)
  const r = await hub.callTool("test_tool", {})
  expect(r.ok).toBe(false)
  expect(r.error).toContain("unreachable")
})

test("makeHubClient: passes args as JSON body", async () => {
  const cfg = { hubURL: "http://test:8080", token: "test-token" } as Pick<Config, "hubURL" | "token">
  let capturedBody = ""
  const fetchImpl: typeof fetch = async (url, init) => {
    capturedBody = init.body ?? ""
    return new Response(JSON.stringify({ text: "ok" }), {
      status: 200,
      headers: { "Content-Type": "application/json" },
    })
  }
  const hub = makeHubClient(cfg, fetchImpl)
  await hub.callTool("test_tool", { key: "value", num: 42 })
  const parsed = JSON.parse(capturedBody)
  expect(parsed.key).toBe("value")
  expect(parsed.num).toBe(42)
})

test("makeHubClient: includes authorization header", async () => {
  const cfg = { hubURL: "http://test:8080", token: "test-token" } as Pick<Config, "hubURL" | "token">
  let capturedHeaders: Record<string, string> = {}
  const fetchImpl: typeof fetch = async (url, init) => {
    capturedHeaders = init.headers ?? {}
    return new Response(JSON.stringify({ text: "ok" }), {
      status: 200,
      headers: { "Content-Type": "application/json" },
    })
  }
  const hub = makeHubClient(cfg, fetchImpl)
  await hub.callTool("test_tool", {})
  expect(capturedHeaders.Authorization).toBe("Bearer test-token")
  expect(capturedHeaders["Content-Type"]).toBe("application/json")
})
