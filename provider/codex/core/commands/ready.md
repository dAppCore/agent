---
name: ready
description: Quick check if work is ready to commit
---

# Ready Check

Quick verification that work is ready to commit.

## Checks

1. No uncommitted changes left behind
2. No debug statements
3. Code is formatted

## Process

```bash
git status --porcelain
core go fmt --check 2>/dev/null || core php fmt --test 2>/dev/null
```

## When to Use

Use `/core:ready` for a quick commit gate.
Use `/core:verify` for the full verification workflow.
