# CoreAgent Vibe Provider

[![License: EUPL-1.2](https://img.shields.io/badge/License-EUPL--1.2-blue.svg)](https://joinup.ec.europa.eu/collection/eupl/eupl-text-eupl-12)

A [Mistral Vibe CLI](https://github.com/mistralai/mistral-vibe) provider plugin that bridges Vibe to the [CoreAgent](https://github.com/host-uk/core-agent) hub, exposing all core-agent MCP tools and enabling fleet coordination through report-home lifecycle hooks.

## Features

- **Full Tool Access**: All 33+ core-agent MCP tools available in Vibe
- **Lifecycle Reporting**: Session start/end/error events reported to CoreAgent
- **Progress Tracking**: Throttled tool execution progress reporting
- **Tool Filtering**: Selectively enable/disable tools via configuration
- **Graceful Degradation**: Tools return error strings instead of throwing

## Tool Categories

| Category | Tools |
|----------|-------|
| **Dispatch** | `dispatch`, `dispatch_remote`, `status`, `status_remote` |
| **Workspace** | `prep_workspace`, `resume`, `watch` |
| **PR/Review** | `create_pr`, `list_prs`, `create_epic`, `review_queue` |
| **Mirror** | `mirror` (Forge → GitHub sync) |
| **Scan** | `scan` (Forge issues) |
| **Brain** | `brain_recall`, `brain_remember`, `brain_forget` |
| **Messaging** | `agent_send`, `agent_inbox`, `agent_conversation` |
| **Plans** | `plan_create`, `plan_read`, `plan_update`, `plan_delete`, `plan_list` |
| **Files** | `file_read`, `file_write`, `file_edit`, `file_delete`, `file_rename`, `file_exists`, `dir_list`, `dir_create` |
| **Language** | `lang_detect`, `lang_list` |

## Installation

### Via npm/npx (Bun required)

```bash
# Install the package
bun add @lthn/core-agent-vibe

# Or from source
cd provider/vibe
bun install
bun run build
```

### Via Vibe Plugin System

Add to your Vibe configuration:

```toml
# ~/.vibe/config.toml

[[providers]]
name = "core-agent"
# Path to the built plugin
path = "/path/to/core-agent/provider/vibe/dist/plugin.js"

# Optional: configure via environment variables
[providers.env]
CORE_HUB_URL = "http://127.0.0.1:9202"
CORE_HUB_TOKEN = "your-hub-token"
CORE_REPORT_TO = "cladius"
```

## Configuration

The plugin is configured via environment variables:

| Variable | Default | Description |
|----------|---------|-------------|
| `CORE_HUB_URL` | `http://127.0.0.1:9202` | Base URL of the core-agent hub MCP plane |
| `CORE_HUB_TOKEN` | (none) | Hub bearer token (or use `CORE_HUB_TOKEN_FILE`) |
| `CORE_HUB_TOKEN_FILE` | (none) | Path to file containing the hub token |
| `CORE_REPORT_TO` | `cladius` | Target agent for report-home messages |
| `CORE_REPORT_WORKSPACE` | (none) | Workspace ID for reporting |
| `CORE_PROGRESS_INTERVAL_MS` | `60000` | Throttle interval for progress reports (ms) |
| `AGENT_NAME` | (none) | Session identity for reporting |
| `CORE_VIBE_ENABLED_TOOLS` | (all) | Comma-separated list of enabled tools (empty = all) |

### Example Configuration

```bash
# Minimal configuration
export CORE_HUB_TOKEN="your-hub-token"

# Full configuration
export CORE_HUB_URL="http://core-agent:9202"
export CORE_HUB_TOKEN="your-hub-token"
export CORE_REPORT_TO="orchestrator"
export CORE_REPORT_WORKSPACE="main-workspace"
export CORE_PROGRESS_INTERVAL_MS="30000"
export AGENT_NAME="vibe-cli"
export CORE_VIBE_ENABLED_TOOLS="dispatch,status,scan,brain_recall"
```

## Usage

### In Vibe CLI

Once installed and configured, Vibe will automatically have access to all core-agent tools:

```bash
# Dispatch a task
vibe "Use the dispatch tool to run a code review"

# Check status
vibe "What's the status of my agent?"

# Scan for issues
vibe "Scan the repository for security issues"

# Recall from brain
vibe "Recall what we know about the auth system"
```

### Programmatic Usage

```typescript
import CoreAgentVibeProvider from "@lthn/core-agent-vibe"

const provider = new CoreAgentVibeProvider()

// Get available tools
const tools = provider.getToolNames()

// Execute a tool
const result = await provider.executeTool("dispatch", {
  repo: "my-repo",
  task: "Fix the bug in auth.ts"
})

// Report lifecycle events
await provider.reportLifecycleEvent({
  type: "session.end",
  properties: { sessionID: "sess-123" }
})

// Direct hub access for advanced usage
const hub = provider.getHubClient()
const response = await hub.callTool("custom_tool", { arg: "value" })
```

## Development

### Build

```bash
cd provider/vibe
bun install
bun run build
```

### Test

```bash
bun test
```

### Type Check

```bash
bun run typecheck
```

## Project Structure

```
provider/vibe/
├── src/
│   ├── config.ts        # Configuration loading
│   ├── hub.ts           # Hub client for MCP tools
│   ├── throttle.ts      # Rate limiting
│   ├── tool_exec.ts     # Tool execution mapping
│   ├── report.ts        # Lifecycle reporting
│   └── plugin.ts        # Main plugin entry point
├── test/
│   ├── config.test.ts
│   ├── hub.test.ts
│   ├── throttle.test.ts
│   ├── tools.test.ts
│   └── report.test.ts
├── package.json
├── tsconfig.json
└── README.md
```

## Architecture

The plugin follows the same pattern as the [opencode provider](https://github.com/host-uk/core-agent/tree/dev/provider/opencode):

1. **Configuration**: Loaded from environment variables with safe defaults
2. **Hub Client**: Communicates with core-agent's MCP HTTP+SSE plane
3. **Tool Mapping**: Static mapping of Vibe tool names to core-agent MCP tools
4. **Reporting**: Session lifecycle and progress events via agent_send
5. **Throttling**: Rate-limited progress reports to prevent flooding

## Contributing

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Run tests: `bun test`
5. Submit a pull request

## License

EUPL-1.2 - See [LICENSE](https://joinup.ec.europa.eu/collection/eupl/eupl-text-eupl-12) for details.
