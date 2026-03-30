# core/agent/flow RFC — YAML-Defined Agent Workflows

> The authoritative spec for the Flow system — declarative, composable, path-addressed agent workflows.
> No code changes needed to improve agent capability. Just YAML + rebuild.

**Package:** `core/agent` (pkg/lib/flow/)
**Repository:** `dappco.re/go/agent`
**Related:** Pipeline Orchestration (core/agent/RFC.pipeline.md)

---

## 1. Overview

Flows are YAML definitions of agent workflows — tasks, prompts, verification steps, security gates. They're composable: flows call other flows. They're path-addressed: the file path IS the semantic meaning.

### 1.1 Design Principle

**Path = semantics.** The same principle as dAppServer's unified path convention:

```
flow/deploy/from/forge.yaml    ← pull from Forge
flow/deploy/to/forge.yaml      ← push to Forge (opposite direction)

flow/workspace/prepare/go.yaml
flow/workspace/prepare/php.yaml
flow/workspace/prepare/devops.yaml
```

An agent navigating by path shouldn't need a README to find the right flow.

### 1.2 Why This Matters

- **Scales without code:** Add a flow YAML, rebuild, done. 20 repos → 200 repos with same effort.
- **Separates what from how:** Flow YAML = intent (what to do). Go code = mechanics (how to do it).
- **Self-healing:** Every problem encountered improves the flow. DevOps lifecycle: hit problem → fix flow → automated forever.
- **Autonomous pipeline:** Issue opened → PR ready for review, without human or orchestrator touching it.

---

## 2. Flow Structure

### 2.1 Basic Flow

```yaml
# flow/verify/go-qa.yaml
name: Go QA
description: Build, test, vet, lint a Go project

steps:
  - name: build
    run: go build ./...

  - name: test
    run: go test ./...

  - name: vet
    run: go vet ./...

  - name: lint
    run: golangci-lint run
```

### 2.2 Composed Flow

Flows call other flows via `flow:` directive:

```yaml
# flow/implement/security-scan.yaml
name: Security Scan Implementation
description: Full lifecycle — prepare, plan, implement, verify, PR

steps:
  - name: prepare
    flow: workspace/prepare/go.yaml

  - name: plan
    agent: spark
    prompt: "Create a security scan implementation plan"

  - name: implement
    agent: codex
    prompt: "Implement the plan"

  - name: verify
    flow: verify/go-qa.yaml

  - name: pr
    flow: pr/to-dev.yaml
```

### 2.3 Agent Steps

Steps can dispatch agents with specific prompts:

```yaml
- name: implement
  agent: codex                    # Agent type
  prompt: |                       # Task prompt
    Read CODEX.md and the RFC at .core/reference/docs/RFC.md.
    Implement the security scan findings.
  template: coding                # Prompt template
  timeout: 30m                    # Max runtime
```

### 2.4 Conditional Steps

```yaml
- name: check-language
  run: cat .core/manifest.yaml | grep language
  output: language

- name: go-verify
  flow: verify/go-qa.yaml
  when: "{{ .language == 'go' }}"

- name: php-verify
  flow: verify/php-qa.yaml
  when: "{{ .language == 'php' }}"
```

---

## 3. Path Convention

### 3.1 Directory Layout

```
pkg/lib/flow/
├── deploy/
│   ├── from/
│   │   └── forge.yaml          # Pull from Forge
│   └── to/
│       ├── forge.yaml          # Push to Forge
│       └── github.yaml         # Push to GitHub
├── implement/
│   ├── security-scan.yaml
│   └── upgrade-deps.yaml
├── pr/
│   ├── to-dev.yaml             # Create PR to dev branch
│   └── to-main.yaml            # Create PR to main branch
├── upgrade/
│   ├── v080-plan.yaml          # Plan v0.8.0 upgrade
│   └── v080-implement.yaml     # Implement v0.8.0 upgrade
├── verify/
│   ├── go-qa.yaml              # Go build+test+vet+lint
│   └── php-qa.yaml             # PHP pest+pint+phpstan
└── workspace/
    └── prepare/
        ├── go.yaml             # Prepare Go workspace
        ├── php.yaml            # Prepare PHP workspace
        ├── ts.yaml             # Prepare TypeScript workspace
        ├── devops.yaml         # Prepare DevOps workspace
        └── secops.yaml         # Prepare SecOps workspace
```

### 3.2 Naming Rules

- **Verbs first:** `deploy/`, `implement/`, `verify/`, `prepare/`
- **Direction explicit:** `from/forge` vs `to/forge`
- **Language suffixed:** `verify/go-qa` vs `verify/php-qa`
- **No abbreviations:** `workspace` not `ws`, `implement` not `impl`

---

## 4. Execution Model

### 4.1 Flow Runner

The Go runner in `pkg/lib/flow/` executes flows:

1. Load YAML flow definition
2. Resolve `flow:` references (recursive)
3. Execute steps sequentially
4. Capture output variables
5. Evaluate `when:` conditions
6. Dispatch agents via Core IPC (runner.dispatch Action)
7. Collect results

### 4.2 CLI Interface

```bash
# Run a flow directly
core-agent run flow pkg/lib/flow/verify/go-qa.yaml

# Dry-run (show what would execute)
core-agent run flow pkg/lib/flow/verify/go-qa.yaml --dry-run

# Run with variables
core-agent run flow pkg/lib/flow/upgrade/v080-implement.yaml --var repo=core/go
```

---

## 5. Composition Patterns

### 5.1 Pipeline (sequential)
```yaml
steps:
  - flow: workspace/prepare/go.yaml
  - flow: verify/go-qa.yaml
  - flow: pr/to-dev.yaml
```

### 5.2 Fan-out (parallel repos)
```yaml
steps:
  - name: upgrade-all
    parallel:
      - flow: upgrade/v080-implement.yaml
        var: { repo: core/go }
      - flow: upgrade/v080-implement.yaml
        var: { repo: core/go-io }
      - flow: upgrade/v080-implement.yaml
        var: { repo: core/go-log }
```

### 5.3 Gate (human approval)
```yaml
steps:
  - flow: implement/security-scan.yaml
  - name: review-gate
    gate: manual
    prompt: "Security scan complete. Review PR before merge?"
  - flow: pr/merge.yaml
```

---

## 6. End State

core-agent CLI runs as a native Forge runner:
1. Forge webhook fires (issue created, PR updated, push event)
2. core-agent picks up the event
3. Selects appropriate flow based on event type + repo config
4. Runs flow → handles full lifecycle
5. No GitHub Actions, no external CI
6. All compute on our hardware
7. Every problem encountered → flow improvement → automated forever

---

## 7. Reference Material

| Resource | Location |
|----------|----------|
| **core/agent** | Flows dispatch agents via Core IPC |
| **core/agent/plugins** | Flows reference agent types (codex, spark, claude) |
| **dAppServer** | Unified path convention = same design principle |
| **core/config** | .core/ convention for workspace detection |

---

## Changelog

- 2026-03-27: Initial RFC promoted from memory + existing flow files. Path-addressed, composable, declarative.
