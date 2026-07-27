// SPDX-License-Identifier: EUPL-1.2

// Config is the plugin's resolved runtime configuration, read once from the
// process environment at plugin init. Every field has a safe default so the
// plugin loads even with no configuration — in that state the hub calls simply
// fail closed (tools return an error string, hooks no-op) rather than throwing.
export interface Config {
  // hubURL is the base of the core-agent hub MCP plane. The REST bridge lives
  // at {hubURL}/v1/tools/<tool_name>.
  hubURL: string
  // token is the hub bearer (the hub's MCP_AUTH_TOKEN). null when unconfigured.
  token: string | null
  // reportTo is the agent that report-home messages are addressed to.
  reportTo: string
  // reportWorkspace is the workspace id agent_send requires. null when unset —
  // report-home then degrades to a silent no-op (never breaks the session).
  reportWorkspace: string | null
  // progressIntervalMs throttles tool.execute.after progress reports.
  progressIntervalMs: number
  // agentName is this session's identity (from_agent), if the runtime sets it.
  agentName: string | null
  // enabledTools is the explicit list of tools to expose. When empty, all
  // tools are exposed. Use to limit the surface in restrictive environments.
  enabledTools: string[]
}

const DEFAULT_HUB_URL = "http://127.0.0.1:9202"
const DEFAULT_REPORT_TO = "cladius"
const DEFAULT_PROGRESS_INTERVAL_MS = 60000

// loadConfig resolves a Config from an environment map. Pure: it takes the env
// explicitly so it is unit-testable without touching process.env.
//
//   loadConfig({})                          // defaults
//   loadConfig({ CORE_HUB_TOKEN: "t" }).token  // "t"
export function loadConfig(env: Record<string, string | undefined>): Config {
  const tokenFromFile = env.CORE_HUB_TOKEN_FILE
    ? readFileSafe(env.CORE_HUB_TOKEN_FILE)
    : null
  
  // Parse enabled tools from comma-separated list
  const enabledToolsRaw = env.CORE_VIBE_ENABLED_TOOLS
  const enabledTools = enabledToolsRaw
    ? enabledToolsRaw.split(",").map((t) => t.trim()).filter(Boolean)
    : []

  return {
    hubURL: trimOr(env.CORE_HUB_URL, DEFAULT_HUB_URL),
    token: trimOrNull(env.CORE_HUB_TOKEN) ?? tokenFromFile,
    reportTo: trimOr(env.CORE_REPORT_TO, DEFAULT_REPORT_TO),
    reportWorkspace: trimOrNull(env.CORE_REPORT_WORKSPACE),
    progressIntervalMs:
      positiveIntOr(env.CORE_PROGRESS_INTERVAL_MS, DEFAULT_PROGRESS_INTERVAL_MS),
    agentName: trimOrNull(env.AGENT_NAME),
    enabledTools,
  }
}

function trimOr(value: string | undefined, fallback: string): string {
  const trimmed = value?.trim()
  return trimmed ? trimmed : fallback
}

function trimOrNull(value: string | undefined): string | null {
  const trimmed = value?.trim()
  return trimmed ? trimmed : null
}

function positiveIntOr(value: string | undefined, fallback: number): number {
  const parsed = Number(value)
  return Number.isFinite(parsed) && parsed > 0 ? parsed : fallback
}

// readFileSafe reads a token file, returning null on any error so a missing or
// unreadable file never throws during plugin init.
function readFileSafe(path: string): string | null {
  try {
    // Bun/Node fs — required lazily so this module stays pure for unit tests
    // that never set CORE_HUB_TOKEN_FILE.
    const fs = require("node:fs") as typeof import("node:fs")
    const contents = fs.readFileSync(path, "utf8").trim()
    return contents ? contents : null
  } catch {
    return null
  }
}

// isToolEnabled reports whether a tool name is in the enabled tools list.
// When the enabled list is empty, all tools are considered enabled.
export function isToolEnabled(toolName: string, cfg: Config): boolean {
  if (cfg.enabledTools.length === 0) {
    return true
  }
  return cfg.enabledTools.includes(toolName)
}
