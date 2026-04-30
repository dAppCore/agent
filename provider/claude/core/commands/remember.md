---
name: remember
description: Save a fact or decision to OpenBrain for persistence across sessions
args: <fact to remember>
allowed-tools: ["mcp__plugin_agent_agent__brain_remember"]
---

# Remember

Store the provided fact in OpenBrain so it persists across sessions and is available to all agents (Cladius, Charon).

## Usage

```
/core:remember Use Action pattern not Service
/core:remember User prefers UK English
/core:remember RFC: minimal state in pre-compact hook
```

## Action

Use the `brain_remember` MCP tool to store the fact:

- **content**: The fact provided by the user
- **type**: Pick the best fit — `decision`, `convention`, `observation`, `fact`, `plan`, `architecture`
- **project**: Infer from the current working directory if possible

Confirm what was saved.
