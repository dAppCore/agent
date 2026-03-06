---
name: deps
description: Show module dependencies
hooks:
  PreCommand:
    - hooks:
        - type: command
          command: "python3 ${CLAUDE_PLUGIN_ROOT}/scripts/deps.py ${TOOL_ARGS}"
---

# /core:deps

Visualize dependencies between modules in the monorepo.

## Usage

`/core:deps` - Show the full dependency tree
`/core:deps <module>` - Show dependencies for a single module
`/core:deps --reverse <module>` - Show what depends on a module
