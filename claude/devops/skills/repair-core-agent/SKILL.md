---
name: repair-core-agent
description: This skill should be used when core-agent is broken, MCP tools aren't responding, dispatch fails, the agent binary is stale, or the user says "fix core-agent", "repair the agent", "core-agent is broken", "MCP not working", "dispatch broken". Diagnoses and guides repair of the core-agent MCP server.
argument-hint: (no arguments needed)
allowed-tools: ["Bash", "Read"]
---

# Repair core-agent

Diagnose and fix core-agent when it's broken.

## Diagnosis Steps

Run these in order, stop at the first failure:

### 1. Is the binary installed?
```bash
which core-agent
```
Should be at `~/.local/bin/core-agent` or `~/go/bin/core-agent`. If BOTH exist, the wrong one might take precedence — check PATH order.

### 2. Does it compile?
```bash
cd /Users/snider/Code/core/agent && go build ./cmd/core-agent/
```
If this fails, there's a code error. Report it and stop.

### 3. Is a stale process running?
```bash
ps aux | grep core-agent | grep -v grep
```
If yes, the user needs to restart it to pick up the new binary.

### 4. Can it start?
```bash
core-agent mcp 2>&1 | head -5
```
Should show subsystem loading messages.

### 5. Are workspaces clean?
```bash
ls /Users/snider/Code/.core/workspace/ 2>/dev/null
```
Should only have `events.jsonl`. Stale workspaces with "running" status but dead PIDs cause phantom slot usage.

### 6. Is agents.yaml readable?
```bash
cat /Users/snider/Code/.core/agents.yaml
```
Check concurrency settings, agent definitions.

## Common Fixes

| Symptom | Fix |
|---------|-----|
| Wrong binary running | Remove stale binary, user reinstalls |
| MCP tools not found | Restart core-agent process |
| Dispatch always queued | Check concurrency limits in agents.yaml |
| Workspaces not prepping | Check workspace template: `ls pkg/lib/workspace/default/` |
| go.work missing from workspace | Rebuild core-agent — template was updated |
| Codex can't find core.Env | Core dep too old — run update-deps skill |

## Important

- Do NOT run `go install` — tell the user to do it
- Do NOT kill processes without asking
- Do NOT delete workspaces without asking
- Report what's wrong, suggest the fix, let the user decide
