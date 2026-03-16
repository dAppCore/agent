---
name: senior-dev-fix
description: Stage 2 of review pipeline — dispatch Senior Developer agent to fix Critical security findings from Stage 1
---

# Senior Developer Fix Stage

Dispatch the **Senior Developer** agent to fix Critical security findings from Stage 1.

## When to Use

Invoked as Stage 2 of `/review:pipeline` ONLY when Stage 1 found Critical issues. Skipped when `--skip=fix` is passed or when there are no Critical findings.

## Agent Persona

Read the Senior Developer persona from:
```
agents/engineering/engineering-senior-developer.md
```

## Dispatch Instructions

1. Read the persona file contents
2. Construct a prompt with the Critical findings from Stage 1
3. Dispatch a subagent with the Agent tool:

```
[Persona content from engineering-senior-developer.md]

## Your Task

Fix the following CRITICAL security issues found during review. Apply the fixes directly to the source files.

### Critical Findings to Fix
[Stage 1 Critical findings — exact file:line, description, recommended fix]

### Rules
- Fix ONLY the Critical issues listed above — do not refactor surrounding code
- Follow existing code style (spacing, braces, naming)
- declare(strict_types=1) in every PHP file
- UK English in all comments and strings
- Run tests after fixing to verify nothing breaks:
  [appropriate test command for the package]

### Output Format

## Fixes Applied

### Fix 1: [Title]
**File**: `path/to/file.php:line`
**Issue**: [What was wrong]
**Change**: [What was changed]

### Fix 2: ...

**Tests**: [PASS/FAIL — test output summary]
```

4. After the subagent completes, re-dispatch Stage 1 (security-review) to verify the fixes resolved the Critical issues
5. If Criticals persist after one fix cycle, report them in the final output rather than looping indefinitely
