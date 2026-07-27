// SPDX-License-Identifier: EUPL-1.2

import type { Config } from "./config.ts"
import type { HubClient } from "./hub.ts"
import type { Throttle } from "./throttle.ts"

// AGENT_SEND is the hub tool report-home messages go through.
const AGENT_SEND = "agent_send"

// LifecycleEvent is the minimal shape we read off an opencode event. opencode
// delivers richer objects; we depend only on these fields.
export interface LifecycleEvent {
  type: string
  properties?: { sessionID?: string; error?: unknown }
}

// buildSendArgs assembles the agent_send arguments for a report. Pure, so the
// argument mapping is unit-testable without a hub. from_agent is omitted when
// the runtime did not set an identity (the hub resolves it server-side).
export function buildSendArgs(
  cfg: Pick<Config, "reportTo" | "reportWorkspace" | "agentName">,
  subject: string,
  content: string,
): Record<string, unknown> {
  const args: Record<string, unknown> = {
    to_agent: cfg.reportTo,
    subject,
    content,
  }
  if (cfg.agentName) args.from_agent = cfg.agentName
  if (cfg.reportWorkspace) args.workspace = cfg.reportWorkspace
  return args
}

// reportLifecycle reports a session lifecycle event home: session.idle → done,
// session.error → BLOCKED. Any other event type is ignored. NEVER throws — a
// failed report must not break the session.
export async function reportLifecycle(
  hub: HubClient,
  cfg: Pick<Config, "reportTo" | "reportWorkspace" | "agentName">,
  event: LifecycleEvent,
): Promise<void> {
  try {
    const sessionID = event.properties?.sessionID ?? "unknown"
    if (event.type === "session.idle") {
      await hub.callTool(AGENT_SEND, buildSendArgs(cfg, "opencode: done", `session ${sessionID} idle`))
      return
    }
    if (event.type === "session.error") {
      const detail = stringifyError(event.properties?.error)
      await hub.callTool(AGENT_SEND, buildSendArgs(cfg, "opencode: BLOCKED", `session ${sessionID}: ${detail}`))
      return
    }
  } catch {
    // silent-on-error invariant: report failures never propagate
  }
}

// reportProgress reports a throttled progress beat after a tool runs. Gated by
// the shared Throttle so noisy tool streams don't flood the orchestrator.
// NEVER throws.
export async function reportProgress(
  hub: HubClient,
  cfg: Pick<Config, "reportTo" | "reportWorkspace" | "agentName">,
  input: { sessionID?: string; tool?: string },
  throttle: Throttle,
  now: number,
): Promise<void> {
  try {
    const sessionID = input.sessionID ?? "unknown"
    if (!throttle.shouldSend(sessionID, now)) {
      return
    }
    const toolName = input.tool ?? "tool"
    await hub.callTool(AGENT_SEND, buildSendArgs(cfg, "opencode: progress", `session ${sessionID} ran ${toolName}`))
  } catch {
    // silent-on-error invariant
  }
}

function stringifyError(err: unknown): string {
  if (err == null) return "unknown error"
  if (typeof err === "string") return err
  if (err instanceof Error) return err.message
  try {
    return JSON.stringify(err)
  } catch {
    return String(err)
  }
}
