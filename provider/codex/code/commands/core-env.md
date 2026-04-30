---
name: /core:env
description: Manage environment configuration
args: [check|diff|sync]
---

# Environment Management

Provides tools for managing `.env` files based on `.env.example`.

## Usage

- `/core:env` - Show current environment variables (with sensitive values masked)
- `/core:env check` - Validate `.env` against `.env.example`
- `/core:env diff` - Show differences between `.env` and `.env.example`
- `/core:env sync` - Add missing variables from `.env.example` to `.env`

## Action

This command is implemented by the following script:

```bash
"${CLAUDE_PLUGIN_ROOT}/scripts/env.sh" "$1"
```
