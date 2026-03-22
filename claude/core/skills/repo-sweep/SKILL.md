---
name: repo-sweep
description: This skill should be used when the user asks to "sweep repos", "audit all repos", "run checks across repos", "batch review", "sweep the ecosystem", or wants to run a task across multiple repositories using agent dispatch with queue monitoring and finding triage. Orchestrates dispatching agents to each repo, monitors the queue drain, and triages findings into actionable issues vs ignorable noise.
version: 0.1.0
---

# Repo Sweep — Dispatched Multi-Repo Audit

Orchestrate a sweep across multiple repositories using `agentic_dispatch`. Dispatch one agent per repo, monitor the queue as agents complete, triage findings, and create issues for actionable results.

## Overview

A repo sweep runs a task across some or all repos in the ecosystem. Each repo gets its own sandboxed agent workspace. The sweep skill handles:

1. **Repo selection** — all repos, filtered by org/language, or explicit list
2. **Dispatch** — queue agents with configurable persona, template, and concurrency
3. **Drain monitoring** — poll `agentic_status` until all complete, process results as they arrive
4. **Triage** — classify findings as actionable (create issue), informational (store to OpenBrain), or noise (ignore)
5. **Reporting** — summary of findings across all repos

## Sweep Process

### Step 1: Select repos

Determine which repos to sweep. Options:

- **All repos**: Query `repos.yaml` at `~/Code/.core/repos.yaml` for the full registry
- **By org**: Filter repos by organisation (core, lthn, ops, host-uk)
- **By language**: Filter by Go, PHP, TypeScript
- **Explicit list**: User provides specific repo names

Parse repos.yaml to get the list:
```bash
cat ~/Code/.core/repos.yaml | grep "name:" | sed 's/.*name: //'
```

Or use `agentic_scan` to find repos with specific labels.

### Step 2: Build the task prompt

Combine the user's sweep goal with a persona and template:

- **Persona**: Controls the agent's lens (e.g., `testing/reality-checker`, `code/senior-developer`, `design/security-developer`)
- **Template**: Controls the approach (e.g., `coding`, `conventions`, `security`, `verify`)
- **Task**: The specific work — provided by the user or from a task bundle (e.g., `code/review`, `code/test-gaps`, `code/dead-code`)

Available personas: check `pkg/lib/persona/` — 116 across 16 domains.
Available templates: `coding`, `conventions`, `security`, `verify`, `default`.
Available tasks: `code/review`, `code/test-gaps`, `code/dead-code`, `code/refactor`, `code/simplifier`, `bug-fix`, `new-feature`, `dependency-audit`, `doc-sync`, `api-consistency`.

### Step 3: Dispatch agents

For each repo, call `agentic_dispatch` with:
- `repo`: the repo name
- `org`: the org (default "core")
- `task`: the constructed task prompt
- `agent`: agent type (default "claude")
- `template`: the prompt template
- `persona`: the persona path

Respect concurrency limits from `config/agents.yaml`. Default: 5 concurrent claude agents. Dispatch in batches — don't flood the queue.

```
For each repo in selected_repos:
    agentic_dispatch(repo=repo, task=task_prompt, agent="claude", template=template, persona=persona)

    // Check concurrency before dispatching next
    status = agentic_status()
    running = count where status == "running"
    if running >= concurrency_limit:
        wait and poll until a slot opens
```

### Step 4: Monitor the drain

Poll `agentic_status` periodically to track progress:

```
while not all_complete:
    status = agentic_status()

    for each completed workspace (new since last check):
        read agent-claude.log from workspace
        triage findings

    report progress: "X/Y complete, Z running"
    wait 30 seconds
```

Read each completed agent's log file at `{workspace_dir}/agent-claude.log`.

### Step 5: Triage findings

For each completed agent's output, classify findings:

**Actionable** (create Forge issue):
- Test failures or missing test coverage
- Security vulnerabilities
- Broken builds or compilation errors
- Convention violations that affect correctness
- Missing documentation for public APIs

**Informational** (store to OpenBrain):
- Code quality observations
- Architecture suggestions
- Performance improvement opportunities
- Patterns observed across repos

**Noise** (ignore):
- Cosmetic style preferences
- Already-known limitations
- Issues in generated/vendored code
- Warnings from deprecated but working code

For actionable findings, use `agentic_create_epic` or create individual issues on Forge via the API.

For informational findings, use `brain_remember` to store observations tagged with the repo name.

### Step 6: Report

Produce a summary:

```markdown
# Sweep Report: {task_description}

## Summary
- Repos swept: X
- Agents completed: Y / X
- Agents failed: Z
- Total findings: N

## Actionable (issues created)
| Repo | Finding | Severity | Issue |
|------|---------|----------|-------|

## Informational (stored to OpenBrain)
| Repo | Observation |
|------|-------------|

## Clean repos (no findings)
- repo-a, repo-b, repo-c
```

Send the report to the user and optionally to Cladius via `agent_send`.

## Configuration

### Default sweep profiles

| Profile | Persona | Template | Task | Use |
|---------|---------|----------|------|-----|
| quality | testing/reality-checker | coding | code/review | Full quality audit |
| security | testing/security-developer | security | — | Security-focused sweep |
| conventions | code/senior-developer | conventions | — | Convention compliance |
| tests | testing/reality-checker | verify | code/test-gaps | Test coverage gaps |
| docs | code/technical-writer | default | doc-sync | Documentation sync |

### Concurrency

Read from `~/Code/core/agent/config/agents.yaml`:
```yaml
concurrency:
  claude: 5
```

Never exceed the configured limit. Leave 1 slot free for interactive work.

## Additional Resources

### Reference Files

For persona and template listings:
- `pkg/lib/persona/` — 116 personas across 16 domains
- `pkg/lib/prompt/` — 5 prompt templates
- `pkg/lib/task/` — 12+ task bundles

### Scripts

- `scripts/list-repos.sh` — Extract repo names from repos.yaml
