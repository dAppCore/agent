// SPDX-License-Identifier: EUPL-1.2

import { test, expect } from "bun:test"
import { makeHubClient, type FetchLike } from "../src/hub.ts"

const cfg = { hubURL: "http://h:9202", token: "tok" }

test("callTool: POSTs the REST bridge with args body + bearer", async () => {
  let seen: { url: string; init: Parameters<FetchLike>[1] } | null = null
  const fakeFetch: FetchLike = async (url, init) => {
    seen = { url, init }
    return new Response(JSON.stringify({ ok: true }), { status: 200 })
  }
  const hub = makeHubClient(cfg, fakeFetch)
  const r = await hub.callTool("agentic_status", { workspace: "w" })

  expect(r.ok).toBe(true)
  expect(seen!.url).toBe("http://h:9202/v1/tools/agentic_status")
  expect(seen!.init.method).toBe("POST")
  expect(seen!.init.headers.Authorization).toBe("Bearer tok")
  expect(seen!.init.headers["Content-Type"]).toBe("application/json")
  expect(JSON.parse(seen!.init.body)).toEqual({ workspace: "w" })
})

test("callTool: prefers a `text` field in the JSON response", async () => {
  const fakeFetch: FetchLike = async () =>
    new Response(JSON.stringify({ text: "human readable" }), { status: 200 })
  const hub = makeHubClient(cfg, fakeFetch)
  const r = await hub.callTool("x", {})
  expect(r.text).toBe("human readable")
})

test("callTool: joins MCP-style content[].text", async () => {
  const fakeFetch: FetchLike = async () =>
    new Response(JSON.stringify({ content: [{ type: "text", text: "a" }, { type: "text", text: "b" }] }), {
      status: 200,
    })
  const hub = makeHubClient(cfg, fakeFetch)
  const r = await hub.callTool("x", {})
  expect(r.text).toBe("a\nb")
})

test("callTool: non-2xx → error result, never throws", async () => {
  const fakeFetch: FetchLike = async () => new Response("nope", { status: 500 })
  const hub = makeHubClient(cfg, fakeFetch)
  const r = await hub.callTool("x", {})
  expect(r.ok).toBe(false)
  expect(r.error).toContain("500")
})

test("callTool: fetch throws → error result, never throws", async () => {
  const fakeFetch: FetchLike = async () => {
    throw new Error("down")
  }
  const hub = makeHubClient(cfg, fakeFetch)
  const r = await hub.callTool("x", {})
  expect(r.ok).toBe(false)
  expect(r.error).toContain("unreachable")
})

test("callTool: no token → error result, fetch never called", async () => {
  let called = false
  const fakeFetch: FetchLike = async () => {
    called = true
    return new Response("")
  }
  const hub = makeHubClient({ hubURL: "http://h", token: null }, fakeFetch)
  const r = await hub.callTool("x", {})
  expect(r.ok).toBe(false)
  expect(called).toBe(false)
})
