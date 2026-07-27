// SPDX-License-Identifier: EUPL-1.2

/**
 * CoreAgent Vibe Provider Plugin
 * 
 * Main exports for the @lthn/core-agent-vibe package.
 */

export { CoreAgentVibeProvider, default } from "./plugin.ts"
export type { Config, HubClient, HubResult, FetchLike, LifecycleEvent, VibeToolName } from "./plugin.ts"
export { loadConfig, isToolEnabled } from "./config.ts"
export { makeHubClient } from "./hub.ts"
export { Throttle } from "./throttle.ts"
export { runTool, runToolDynamic, getToolList, getToolDescription, TOOL_MAP } from "./tool_exec.ts"
export { buildSendArgs, reportLifecycle, reportProgress, reportToolStart } from "./report.ts"
