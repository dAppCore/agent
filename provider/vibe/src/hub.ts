// SPDX-License-Identifier: EUPL-1.2

import type { Config } from "./config.ts"

// HubResult is the outcome of one hub tool call. It never represents failure as
// a thrown error — callers (tools, hooks) depend on this so a down hub can
// never break a Vibe session.
export interface HubResult {
  ok: boolean
  text?: string
  error?: string
}

// HubClient calls a single core-agent capability by its hub tool name.
export interface HubClient {
  callTool(name: string, args: Record<string, unknown>): Promise<HubResult>
}

// FetchLike is the subset of fetch the client uses — injectable for tests.
export type FetchLike = (
  url: string,
  init: { method: string; headers: Record<string, string>; body: string },
) => Promise<Response>

// makeHubClient builds a HubClient over the hub's stateless REST bridge:
//   POST {hubURL}/v1/tools/<name>  body=<args>  Authorization: Bearer <token>
// The response body (the tool's JSON output) becomes the result text. Every
// failure mode — no token, non-2xx, network throw — resolves to { ok: false }.
//
//   const hub = makeHubClient(cfg)
//   const r = await hub.callTool("agentic_status", { workspace: "w" })
export function makeHubClient(
  cfg: Pick<Config, "hubURL" | "token">,
  fetchImpl: FetchLike = fetch as unknown as FetchLike,
): HubClient {
  return {
    async callTool(name, args): Promise<HubResult> {
      if (!cfg.token) {
        return { ok: false, error: "hub token not configured (set CORE_HUB_TOKEN or CORE_HUB_TOKEN_FILE)" }
      }
      const url = `${stripTrailingSlash(cfg.hubURL)}/v1/tools/${name}`
      try {
        const res = await fetchImpl(url, {
          method: "POST",
          headers: {
            Authorization: `Bearer ${cfg.token}`,
            "Content-Type": "application/json",
          },
          body: JSON.stringify(args ?? {}),
        })
        const text = await res.text()
        if (!res.ok) {
          return { ok: false, error: `hub ${res.status}: ${text}` }
        }
        return { ok: true, text: extractText(text) }
      } catch (err) {
        return { ok: false, error: `hub unreachable: ${String(err)}` }
      }
    },
  }
}

function stripTrailingSlash(url: string): string {
  return url.endsWith("/") ? url.slice(0, -1) : url
}

// extractText returns the most useful human/string view of a tool response.
// The REST bridge returns the tool's JSON output; when that JSON carries a
// `text` field or an MCP-style `content[].text`, prefer it; otherwise return
// the raw body.
function extractText(body: string): string {
  try {
    const parsed = JSON.parse(body) as unknown
    if (parsed && typeof parsed === "object") {
      const obj = parsed as Record<string, unknown>
      if (typeof obj.text === "string") return obj.text
      const content = obj.content
      if (Array.isArray(content)) {
        const joined = content
          .map((part) =>
            part && typeof part === "object" && typeof (part as Record<string, unknown>).text === "string"
              ? ((part as Record<string, unknown>).text as string)
              : "",
          )
          .filter(Boolean)
          .join("\n")
        if (joined) return joined
      }
    }
  } catch {
    // not JSON — fall through to the raw body
  }
  return body
}
