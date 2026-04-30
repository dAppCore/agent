---
name: sweep
description: Dispatch a batch audit across multiple repos
arguments:
  - name: template
    description: Audit template (conventions, security)
    default: conventions
  - name: agent
    description: Agent type for the sweep
    default: codex
  - name: repos
    description: Comma-separated repos to include (default: all Go repos)
---

Run a batch conventions or security audit across the ecosystem.

1. If repos are not specified, find all repos under the configured workspace root that match the target language and template
2. For each repo, call `agentic_dispatch` with:
   - repo
   - task: `"{template} audit - UK English, error handling, interface checks, import aliasing"`
   - agent: `$ARGUMENTS.agent`
   - template: `$ARGUMENTS.template`
3. Report how many were dispatched versus queued
4. Point the user to `/core:status` and `/core:review` for follow-up
