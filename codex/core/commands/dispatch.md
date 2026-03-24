---
name: dispatch
description: Dispatch a subagent to work on a task in a sandboxed workspace
arguments:
  - name: repo
    description: Target repo (e.g. go-io, go-scm, mcp)
    required: true
  - name: task
    description: What the agent should do
    required: true
  - name: agent
    description: Agent type (claude, gemini, codex)
    default: codex
  - name: template
    description: Prompt template (coding, conventions, security)
    default: coding
  - name: plan
    description: Plan template (bug-fix, code-review, new-feature, refactor, feature-port)
  - name: persona
    description: Persona slug (e.g. code/backend-architect)
---

Dispatch a subagent to work on `$ARGUMENTS.repo` with task: `$ARGUMENTS.task`

Use the core-agent MCP tool `agentic_dispatch` with:
- repo: `$ARGUMENTS.repo`
- task: `$ARGUMENTS.task`
- agent: `$ARGUMENTS.agent`
- template: `$ARGUMENTS.template`
- plan_template: `$ARGUMENTS.plan` if provided
- persona: `$ARGUMENTS.persona` if provided

After dispatching, report the workspace dir, PID, and whether the task was queued or started immediately.
