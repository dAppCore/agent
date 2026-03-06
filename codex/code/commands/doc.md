---
name: doc
description: Auto-generate documentation from code.
hooks:
  PostToolUse:
    - matcher: "Tool"
      hooks:
        - type: command
          command: "${CLAUDE_PLUGIN_ROOT}/scripts/doc.sh"
---

# Documentation Generator

This command generates documentation from your codebase.

## Usage

`/core:doc <type> <name>`

## Subcommands

- **class <ClassName>**: Document a single class.
- **api**: Generate OpenAPI spec for the project.
- **changelog**: Generate a changelog from git commits.
