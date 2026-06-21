// SPDX-License-Identifier: EUPL-1.2

import type { HubClient } from "./hub.ts"

// TOOL_MAP maps each Vibe-facing tool name to the core-agent hub tool it
// bridges to. This is the v1 capability subset (RFC §7): dispatch + status +
// scan + the two brain verbs + messaging + plans + files + language.
export const TOOL_MAP = {
  // Dispatch
  dispatch: "agentic_dispatch",
  dispatch_remote: "agentic_dispatch_remote",
  status: "agentic_status",
  status_remote: "agentic_status_remote",
  
  // Workspace
  prep_workspace: "agentic_prep_workspace",
  resume: "agentic_resume",
  watch: "agentic_watch",
  
  // PR/Review
  create_pr: "agentic_create_pr",
  list_prs: "agentic_list_prs",
  create_epic: "agentic_create_epic",
  review_queue: "agentic_review_queue",
  
  // Mirror
  mirror: "agentic_mirror",
  
  // Scan
  scan: "agentic_scan",
  
  // Brain
  brain_recall: "brain_recall",
  brain_remember: "brain_remember",
  brain_forget: "brain_forget",
  
  // Messaging
  agent_send: "agent_send",
  agent_inbox: "agent_inbox",
  agent_conversation: "agent_conversation",
  
  // Plans
  plan_create: "agentic_plan_create",
  plan_read: "agentic_plan_read",
  plan_update: "agentic_plan_update",
  plan_delete: "agentic_plan_delete",
  plan_list: "agentic_plan_list",
  
  // Files
  file_read: "file_read",
  file_write: "file_write",
  file_edit: "file_edit",
  file_delete: "file_delete",
  file_rename: "file_rename",
  file_exists: "file_exists",
  dir_list: "dir_list",
  dir_create: "dir_create",
  
  // Language
  lang_detect: "lang_detect",
  lang_list: "lang_list",
} as const

// VibeToolName is the type of valid Vibe-facing tool names.
export type VibeToolName = keyof typeof TOOL_MAP

// runTool calls one hub tool and renders a string result for the model. It
// never throws: a hub failure becomes a readable error string, so a tool call
// degrades gracefully instead of crashing the session.
//
//   await runTool(hub, "dispatch", { repo: "r", task: "t" })
export async function runTool(
  hub: HubClient,
  vibeName: VibeToolName,
  args: Record<string, unknown>,
): Promise<string> {
  const mcpName = TOOL_MAP[vibeName]
  const r = await hub.callTool(mcpName, args)
  if (r.ok) {
    return r.text ?? ""
  }
  return `${vibeName} failed: ${r.error ?? "unknown error"}`
}

// runToolDynamic calls a hub tool by its raw name (for tools not in TOOL_MAP).
// Useful for custom or future tools. Never throws.
export async function runToolDynamic(
  hub: HubClient,
  name: string,
  args: Record<string, unknown>,
): Promise<string> {
  const r = await hub.callTool(name, args)
  if (r.ok) {
    return r.text ?? ""
  }
  return `${name} failed: ${r.error ?? "unknown error"}`
}

// getToolList returns the list of all available Vibe tool names.
export function getToolList(): VibeToolName[] {
  return Object.keys(TOOL_MAP) as VibeToolName[]
}

// getToolDescription returns a description for a tool, useful for tool discovery.
export function getToolDescription(name: VibeToolName): string {
  const descriptions: Record<VibeToolName, string> = {
    dispatch: "Dispatch a task to the core-agent system",
    dispatch_remote: "Dispatch a task to a remote core-agent host",
    status: "Check the status of an agent or task",
    status_remote: "Check the status of a remote agent or task",
    prep_workspace: "Prepare a workspace for agent operations",
    resume: "Resume a paused or interrupted session",
    watch: "Watch for changes in a workspace",
    create_pr: "Create a pull request from completed work",
    list_prs: "List active pull requests",
    create_epic: "Create an epic for tracking multiple tasks",
    review_queue: "Check the code review queue",
    mirror: "Mirror repositories between forge and GitHub",
    scan: "Scan repositories for issues",
    brain_recall: "Recall information from the OpenBrain knowledge base",
    brain_remember: "Store information in the OpenBrain knowledge base",
    brain_forget: "Remove information from the OpenBrain knowledge base",
    agent_send: "Send a message to another agent",
    agent_inbox: "Check your agent inbox for messages",
    agent_conversation: "Start or continue a conversation with another agent",
    plan_create: "Create a new plan",
    plan_read: "Read a plan",
    plan_update: "Update an existing plan",
    plan_delete: "Delete a plan",
    plan_list: "List all plans",
    file_read: "Read a file",
    file_write: "Write a file",
    file_edit: "Edit a file",
    file_delete: "Delete a file",
    file_rename: "Rename a file",
    file_exists: "Check if a file exists",
    dir_list: "List directory contents",
    dir_create: "Create a directory",
    lang_detect: "Detect the language of text",
    lang_list: "List available languages",
  }
  return descriptions[name] ?? `core-agent tool: ${name}`
}
