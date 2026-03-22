---
name: review-pipeline
description: Run the multi-stage review pipeline — security, fix, simplify, architecture, verify
arguments:
  - name: target
    description: Directory or repo to review
    default: .
  - name: stages
    description: Comma-separated stages (security,fix,simplify,architecture,verify)
    default: security,fix,simplify,architecture,verify
  - name: skip
    description: Stages to skip
---

# Review Pipeline

Multi-stage code review with specialist agents at each stage.

## Stages

### 1. Security Review
Dispatch agent with `secops/developer` persona:
```bash
cat ~/Code/core/agent/pkg/prompts/lib/persona/secops/developer.md
```
Task: scan for OWASP top 10, injection, path traversal, race conditions.
Report findings as CRITICAL/HIGH/MEDIUM/LOW with file:line.

### 2. Fix (conditional)
Only runs if Stage 1 found CRITICAL issues.
Dispatch agent with task from `lib/task/code/review.md`.
Fix ONLY critical findings, nothing else.

### 3. Simplify
Dispatch code-simplifier agent:
```bash
cat ~/Code/core/agent/claude/core/agents/agent-task-code-simplifier.md
```
Reduce complexity, remove dead code, improve naming.

### 4. Architecture Review
Dispatch with `code/backend-architect` persona:
```bash
cat ~/Code/core/agent/pkg/prompts/lib/persona/code/backend-architect.md
```
Check patterns, dependency direction, lifecycle correctness.

### 5. Verify
```bash
cd $ARGUMENTS.target
go build ./... 2>&1
go vet ./... 2>&1
go test ./... -count=1 -timeout 60s 2>&1 | tail -20
```

## Flow Control

- If `--skip=fix` → skip Stage 2
- If Stage 1 has 0 criticals → skip Stage 2 automatically
- If Stage 5 fails → report and stop
- Each stage output feeds into the next as context

## Output

```markdown
## Review Pipeline: $ARGUMENTS.target

| Stage | Status | Findings |
|-------|--------|----------|
| Security | PASS/FAIL | X critical, Y high |
| Fix | APPLIED/SKIPPED | N fixes |
| Simplify | DONE | N changes |
| Architecture | PASS/FAIL | X violations |
| Verify | PASS/FAIL | build + vet + test |
```
