---
name: security-review
description: Stage 1 of review pipeline — dispatch Security Engineer agent for threat analysis, injection review, and tenant isolation checks on code changes
---

# Security Review Stage

Dispatch the **Security Engineer** agent to perform a read-only security review of code changes.

## When to Use

This skill is invoked as Stage 1 of the `/review:pipeline` command. It can also be triggered standalone via `/review:pipeline --stage=security`.

## Agent Persona

Read the Security Engineer persona from:
```
agents/engineering/engineering-security-engineer.md
```

## Dispatch Instructions

1. Read the persona file contents
2. Read the diff and list of changed files
3. Dispatch a subagent with the Agent tool using this prompt structure:

```
[Persona content from engineering-security-engineer.md]

## Your Task

Perform a security-focused code review of the following changes. This is a READ-ONLY review — do not modify any files.

### Changed Files
[List of changed files]

### Diff
[Full diff content]

### Focus Areas
- Arbitrary code execution vectors
- Method/class injection from DB or config values
- Tenant isolation (BelongsToWorkspace on all tenant-scoped models)
- Input validation in Action handle() methods
- Namespace safety (allowlists where class names come from external sources)
- Error handling (no silent swallowing, no stack trace leakage)
- Secrets in code (API keys, credentials, .env values)

### Output Format

Produce findings in this exact format:

## Security Review Findings

### CRITICAL
- **file:line** — [Title]: [Description]. **Attack vector**: [How]. **Fix**: [What to change]

### HIGH
- **file:line** — [Title]: [Description]. **Fix**: [What to change]

### MEDIUM
- **file:line** — [Title]: [Description]. **Fix**: [What to change]

### LOW
- **file:line** — [Title]: [Description]

### Positive Controls
[Things done well — allowlists, guards, scoping]

**Summary**: X critical, Y high, Z medium, W low
```

4. Return the subagent's findings as the stage output
