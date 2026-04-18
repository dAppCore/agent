---
name: sweep
description: Dispatch a batch audit across all Go repos in the ecosystem
arguments:
  - name: template
    description: Audit template (conventions, security)
    default: conventions
  - name: agent
    description: Agent type for the sweep
    default: gemini
  - name: repos
    description: Comma-separated repos to include (default: all Go repos)
---

Run a batch conventions or security audit across the Go ecosystem.

1. If repos not specified, find all Go repos in ~/Code/core/ that have a go.mod
2. For each repo, call `mcp__plugin_agent_agent__agentic_dispatch` with:
   - repo: {repo name}
   - task: "{template} audit - UK English, error handling, interface checks, import aliasing"
   - agent: $ARGUMENTS.agent
   - template: $ARGUMENTS.template
3. Report how many were dispatched vs queued
4. Tell the user they can check progress with `/core:status` and review results with `/core:review`
