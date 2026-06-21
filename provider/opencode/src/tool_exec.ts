// SPDX-License-Identifier: EUPL-1.2

import type { HubClient } from "./hub.ts"

// TOOL_MAP maps each opencode-facing tool name to the core-agent hub tool it
// bridges to. This is the v1 capability subset (RFC §7): dispatch + status +
// scan + the two brain verbs.
export const TOOL_MAP = {
  dispatch: "agentic_dispatch",
  status: "agentic_status",
  scan: "agentic_scan",
  brain_recall: "brain_recall",
  brain_remember: "brain_remember",
} as const

export type OpencodeToolName = keyof typeof TOOL_MAP

// runTool calls one hub tool and renders a string result for the model. It
// never throws: a hub failure becomes a readable error string, so a tool call
// degrades gracefully instead of crashing the session.
//
//   await runTool(hub, "agentic_status", { workspace: "w" })
export async function runTool(
  hub: HubClient,
  mcpName: string,
  args: Record<string, unknown>,
): Promise<string> {
  const r = await hub.callTool(mcpName, args)
  if (r.ok) {
    return r.text ?? ""
  }
  return `${mcpName} failed: ${r.error ?? "unknown error"}`
}
