---
name: reality-check
description: Stage 5 of review pipeline — dispatch Reality Checker agent as final gate with evidence-based verdict
---

# Reality Check Stage (Final Gate)

Dispatch the **Reality Checker** agent as the final review gate. Defaults to NEEDS WORK.

## When to Use

Invoked as Stage 5 (final stage) of `/review:pipeline`. Can be run standalone via `/review:pipeline --stage=reality`.

## Agent Persona

Read the Reality Checker persona from:
```
agents/testing/testing-reality-checker.md
```

## Dispatch Instructions

1. Read the persona file contents
2. Gather ALL prior stage findings into a single context block
3. Dispatch a subagent:

```
[Persona content from testing-reality-checker.md]

## Your Task

You are the FINAL GATE. Review all prior stage findings and produce an evidence-based verdict. Default to NEEDS WORK.

### Prior Stage Findings

#### Stage 1: Security Review
[Stage 1 output]

#### Stage 2: Fixes Applied
[Stage 2 output, or "Skipped"]

#### Stage 3: Test Analysis
[Stage 3 output]

#### Stage 4: Architecture Review
[Stage 4 output]

### Changed Files
[List of changed files]

### Your Assessment

1. **Cross-reference all findings** — do security fixes have tests? Do architecture violations have security implications?
2. **Verify evidence** — are test results real (actual command output) or claimed?
3. **Check for gaps** — what did previous stages miss?
4. **Apply your FAIL triggers** — fantasy assessments, missing evidence, architecture violations

### Output Format

## Final Verdict

**Status**: READY / NEEDS WORK / FAILED
**Quality Rating**: C+ / B- / B / B+

### Evidence Summary
| Check | Status | Evidence |
|-------|--------|----------|
| Tests pass | YES/NO | [Command + output] |
| Lint clean | YES/NO | [Command + output] |
| Security issues resolved | YES/NO | [Remaining count] |
| Architecture correct | YES/NO | [Violation count] |
| Tenant isolation verified | YES/NO | [Specific check] |
| UK English | YES/NO | [Violations found] |
| Test coverage of changes | X/Y | [Gap count] |

### Outstanding Issues
1. **[CRITICAL/IMPORTANT/MINOR]**: file:line — [Issue]
2. ...

### Required Before Merge
1. [Specific action with file path]
2. ...

### What's Done Well
[Positive findings from all stages]

---
**Reviewer**: Reality Checker
**Date**: [Date]
**Re-review required**: YES/NO
```

4. Return the subagent's verdict as the final pipeline output
