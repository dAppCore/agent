---
name: remember
description: Save a fact or decision to OpenBrain for persistence across sessions
args: <fact to remember>
---

# Remember

Store the provided fact in OpenBrain so it persists across sessions and is available to other agents.

Use the core-agent MCP tool `brain_remember` with:

- `content`: the fact provided by the user
- `type`: best fit from `decision`, `convention`, `observation`, `fact`, `plan`, or `architecture`
- `project`: infer from the current working directory when possible

Confirm what was saved.
