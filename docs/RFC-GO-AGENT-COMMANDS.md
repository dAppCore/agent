# core-agent — Commands

> CLI commands and MCP tool registrations.

## CLI Commands

```
core-agent [command]
```

| Command | Purpose |
|---------|---------|
| `version` | Print version |
| `check` | Health check |
| `env` | Show environment |
| `run/task` | Run a single agent task |
| `run/orchestrator` | Run the orchestrator daemon |
| `prep` | Prepare workspace without spawning |
| `status` | Show workspace status |
| `prompt` | Build/preview agent prompt |
| `extract` | Extract data from agent output |
| `workspace/list` | List agent workspaces |
| `workspace/clean` | Clean completed/failed workspaces |
| `workspace/dispatch` | Dispatch agent to workspace |
| `issue/get` | Get Forge issue by number |
| `issue/list` | List Forge issues |
| `issue/comment` | Comment on Forge issue |
| `issue/create` | Create Forge issue |
| `pr/get` | Get Forge PR by number |
| `pr/list` | List Forge PRs |
| `pr/merge` | Merge Forge PR |
| `repo/get` | Get Forge repo info |
| `repo/list` | List Forge repos |
| `repo/sync` | Fetch and optionally reset a local repo from origin |
| `mcp` | Start MCP server (stdio) |
| `serve` | Start HTTP/API server |

## MCP Tools (via `core-agent mcp`)

### agentic (PrepSubsystem.RegisterTools)

- `agentic_dispatch` — dispatch a subagent to a sandboxed workspace
- `agentic_prep_workspace` — prepare workspace without spawning
- `agentic_status` — list agent workspaces and their status
- `agentic_watch` — watch running agents until completion
- `agentic_resume` — resume a blocked agent
- `agentic_review_queue` — review completed workspaces
- `agentic_scan` — scan Forge for actionable issues
- `agentic_mirror` — mirror repos between remotes
- `agentic_plan_create` / `plan_read` / `plan_update` / `plan_delete` / `plan_list`
- `agentic_create_pr` — create PR from agent workspace
- `agentic_create_epic` — create epic with child issues
- `agentic_dispatch_start` / `dispatch_shutdown` / `dispatch_shutdown_now`
- `agentic_dispatch_remote` / `agentic_status_remote`

### brain (DirectSubsystem.RegisterTools)

- `brain_recall` — search OpenBrain memories
- `brain_remember` — store a memory
- `brain_forget` — remove a memory

### brain (DirectSubsystem.RegisterMessagingTools)

- `agent_send` — send message to another agent
- `agent_inbox` — check incoming messages
- `agent_conversation` — view conversation history

### monitor (Subsystem.RegisterTools)

- Exposes agent workspace status as MCP resource

### File operations (via core-mcp)

- `file_read` / `file_write` / `file_edit` / `file_delete` / `file_rename` / `file_exists`
- `dir_list` / `dir_create`
- `lang_detect` / `lang_list`
