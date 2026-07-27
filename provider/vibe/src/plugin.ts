// SPDX-License-Identifier: EUPL-1.2

/**
 * CoreAgent Vibe Provider Plugin
 * 
 * This plugin bridges Mistral Vibe CLI to the CoreAgent hub, exposing:
 * - All core-agent MCP tools (dispatch, status, scan, brain, messaging, plans, files, language)
 * - Report-home lifecycle hooks for fleet coordination
 * - Throttled progress reporting
 * 
 * Configuration via environment variables:
 * - CORE_HUB_URL: hub base URL (default: http://127.0.0.1:9202)
 * - CORE_HUB_TOKEN: hub bearer token (or CORE_HUB_TOKEN_FILE for file-based)
 * - CORE_REPORT_TO: report target agent (default: cladius)
 * - CORE_REPORT_WORKSPACE: workspace ID for reporting
 * - CORE_PROGRESS_INTERVAL_MS: throttle interval (default: 60000)
 * - AGENT_NAME: session identity
 * - CORE_VIBE_ENABLED_TOOLS: comma-separated list of enabled tools (empty = all)
 */

import type { Config, isToolEnabled } from "./config.ts"
import { loadConfig } from "./config.ts"
import type { HubClient } from "./hub.ts"
import { makeHubClient } from "./hub.ts"
import type { Throttle } from "./throttle.ts"
import { runTool, runToolDynamic, getToolList, getToolDescription, type VibeToolName } from "./tool_exec.ts"
import { reportLifecycle, reportProgress, reportToolStart, type LifecycleEvent } from "./report.ts"

// PluginConfig is the resolved configuration for the plugin instance.
interface PluginConfig extends Config {
  hub: HubClient
  throttle: Throttle
}

// CoreAgentVibeProvider is the main plugin class that provides CoreAgent
// capabilities to Vibe CLI.
export class CoreAgentVibeProvider {
  private cfg: PluginConfig
  private initialised = false

  constructor() {
    // Configuration is loaded lazily on first use
    this.cfg = this.loadPluginConfig()
  }

  private loadPluginConfig(): PluginConfig {
    const env = process.env as Record<string, string | undefined>
    const config = loadConfig(env)
    return {
      ...config,
      hub: makeHubClient(config),
      throttle: new Throttle(config.progressIntervalMs),
    }
  }

  /**
   * Initialise the plugin. Called by Vibe when the plugin is loaded.
   */
  async initialise(): Promise<void> {
    if (this.initialised) return
    this.initialised = true
    
    // Validate configuration
    if (!this.cfg.token) {
      console.warn("[core-agent-vibe] Warning: CORE_HUB_TOKEN not configured. Tools will return errors.")
    }
  }

  /**
   * Get the list of available tool names.
   */
  getToolNames(): VibeToolName[] {
    return getToolList()
  }

  /**
   * Get the description for a specific tool.
   */
  getToolDescription(name: VibeToolName): string {
    return getToolDescription(name)
  }

  /**
   * Check if a tool is enabled.
   */
  isToolEnabled(name: VibeToolName): boolean {
    return isToolEnabled(name, this.cfg)
  }

  /**
   * Execute a CoreAgent tool.
   * This is the main entry point called by Vibe when a tool is invoked.
   */
  async executeTool(name: string, args: Record<string, unknown>): Promise<string> {
    // Lazy initialisation
    await this.initialise()

    // Check if tool is enabled
    if (getToolList().includes(name as VibeToolName)) {
      if (!this.isToolEnabled(name as VibeToolName)) {
        return `Tool ${name} is disabled. Enable it via CORE_VIBE_ENABLED_TOOLS.`
      }
    }

    // Report tool start
    await this.reportToolStart(name, args)

    // Execute the tool
    try {
      if (getToolList().includes(name as VibeToolName)) {
        return await runTool(this.cfg.hub, name as VibeToolName, args)
      }
      // Dynamic tool execution for tools not in the static map
      return await runToolDynamic(this.cfg.hub, name, args)
    } catch (err) {
      // Should never throw, but just in case
      return `${name} failed: ${String(err)}`
    }
  }

  /**
   * Report a tool start for monitoring.
   */
  private async reportToolStart(name: string, args: Record<string, unknown>): Promise<void> {
    try {
      const sessionID = this.getSessionID()
      await reportToolStart(this.cfg.hub, this.cfg, { sessionID, tool: name, args })
    } catch {
      // Silent failure
    }
  }

  /**
   * Report a lifecycle event.
   * Called by Vibe hooks for session lifecycle events.
   */
  async reportLifecycleEvent(event: LifecycleEvent): Promise<void> {
    await this.initialise()
    await reportLifecycle(this.cfg.hub, this.cfg, event)
  }

  /**
   * Report progress after tool execution.
   * Called by Vibe hooks after each tool execution.
   */
  async reportProgressEvent(sessionID: string, tool: string): Promise<void> {
    await this.initialise()
    const now = Date.now()
    await reportProgress(this.cfg.hub, this.cfg, { sessionID, tool, agent: "vibe" }, this.cfg.throttle, now)
  }

  /**
   * Get the current session ID from the environment.
   * Vibe sets this in the environment for hooks.
   */
  getSessionID(): string | undefined {
    return process.env.VIBE_SESSION_ID
  }

  /**
   * Get the current agent name.
   */
  getAgentName(): string | null {
    return this.cfg.agentName
  }

  /**
   * Get the hub configuration for advanced usage.
   */
  getHubClient(): HubClient {
    return this.cfg.hub
  }

  /**
   * Reset the throttle state (useful for testing).
   */
  resetThrottle(): void {
    this.cfg.throttle.clear()
  }
}

// Default export for Vibe plugin system
export default CoreAgentVibeProvider

// Named export for direct import
export { CoreAgentVibeProvider }

// Re-export types for external usage
export type { Config, HubClient, HubResult, FetchLike, LifecycleEvent, VibeToolName }
export { loadConfig, makeHubClient, runTool, runToolDynamic, getToolList, getToolDescription, reportLifecycle, reportProgress }
