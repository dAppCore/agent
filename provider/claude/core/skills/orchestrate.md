---
name: orchestrate
description: Run the full agent pipeline — plan, dispatch, monitor, review, fix, re-review, merge. Works locally without MCP.
arguments:
  - name: repo
    description: Target repo (e.g. go, go-process, mcp)
    required: true
  - name: goal
    description: What needs to be achieved
    required: true
  - name: agent
    description: Agent type (claude:haiku, claude:sonnet, gemini, codex)
    default: claude:haiku
  - name: stages
    description: Comma-separated stages to run (plan,dispatch,review,fix,verify)
    default: plan,dispatch,review,fix,verify
---

# Agent Orchestration Pipeline

Run the full dispatch → review → fix → verify cycle for `$ARGUMENTS.repo`.

## Stage 1: Plan

Break `$ARGUMENTS.goal` into discrete tasks. For each task:
- Determine the best persona from `lib/persona/`
- Select the right prompt template from `lib/prompt/`
- Choose a task plan from `lib/task/` if one fits

List tasks using the prompts library:
```bash
# Available personas
find ~/Code/core/agent/pkg/prompts/lib/persona -name "*.md" | sed 's|.*/lib/persona/||;s|\.md$||'

# Available task plans
find ~/Code/core/agent/pkg/prompts/lib/task -name "*.md" -o -name "*.yaml" | sed 's|.*/lib/task/||;s|\.(md|yaml)$||'

# Available prompt templates
find ~/Code/core/agent/pkg/prompts/lib/prompt -name "*.md" | sed 's|.*/lib/prompt/||;s|\.md$||'

# Available flows
find ~/Code/core/agent/pkg/prompts/lib/flow -name "*.md" | sed 's|.*/lib/flow/||;s|\.md$||'
```

Output a task list with: task name, persona, template, estimated complexity.

## Stage 2: Dispatch

For each task from Stage 1, dispatch an agent. Prefer MCP tools if available:
```
mcp__plugin_agent_agent__agentic_dispatch(repo, task, agent, template, persona)
```

If MCP is unavailable, dispatch locally:
```bash
cd ~/Code/core/$ARGUMENTS.repo
claude --dangerously-skip-permissions -p "[persona content]

$TASK_DESCRIPTION" --model $MODEL
```

Track dispatched tasks: workspace dir, PID, status.

## Stage 3: Review

After agents complete, review their output:
```bash
# Check workspace status
ls ~/Code/.core/workspace/$REPO-*/

# Read agent logs
cat ~/Code/.core/workspace/$WORKSPACE/agent-*.log

# Check for commits
cd ~/Code/.core/workspace/$WORKSPACE/src && git log --oneline -5
```

Run the code-review agent on changes:
```
Read lib/task/code/review.md and dispatch review agent
```

## Stage 4: Fix

If review finds issues, dispatch a fix agent:
- Use `lib/task/code/review.md` findings as input
- Use `secops/developer` persona for security fixes
- Use `code/backend-architect` persona for structural fixes

## Stage 5: Verify

Final check:
```bash
# Build
cd ~/Code/.core/workspace/$WORKSPACE/src && go build ./...

# Vet
go vet ./...

# Run targeted tests if they exist
go test ./... -count=1 -timeout 60s 2>&1 | tail -20
```

If verify passes → report success.
If verify fails → report failures and stop.

## Output

```markdown
## Orchestration Report: $ARGUMENTS.repo

### Goal
$ARGUMENTS.goal

### Tasks Dispatched
| # | Task | Agent | Status |
|---|------|-------|--------|

### Review Findings
[Summary from Stage 3]

### Fixes Applied
[Summary from Stage 4]

### Verification
[PASS/FAIL from Stage 5]

### Next Steps
[Any remaining work]
```
