---
name: code-review
description: Perform code review on staged changes or PRs
args: [commit-range|--pr=N|--security]
---

# Code Review

Perform a thorough code review of the specified changes.

## Arguments

- No args: Review staged changes
- `HEAD~3..HEAD`: Review last 3 commits
- `--pr=123`: Review PR #123
- `--security`: Focus on security issues

## Process

1. Gather changes from the requested diff target
2. Analyse each changed file for correctness, security, maintainability, and test gaps
3. Report findings with clear severity and file references

## Review Checklist

| Category | Checks |
|----------|--------|
| Correctness | Logic errors, edge cases, error handling |
| Security | Injection, XSS, hardcoded secrets, CSRF |
| Performance | N+1 queries, unnecessary loops, large allocations |
| Maintainability | Naming, structure, complexity |
| Tests | Coverage gaps, missing assertions |

## Output Format

```markdown
## Code Review: [title]

### Critical
- **file:line** - Issue description

### Warning
- **file:line** - Issue description

### Suggestions
- **file:line** - Improvement idea

---
**Summary**: X critical, Y warnings, Z suggestions
```
