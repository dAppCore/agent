---
name: status
description: Show CI status for current branch
---

# CI Status

Show GitHub Actions status for the current branch.

## Usage

```
/ci:status
/ci:status --all      # All recent runs
/ci:status --branch X # Specific branch
```

## Commands

```bash
# Current branch status
gh run list --branch $(git branch --show-current) --limit 5

# Get details of latest run
gh run view --log-failed

# Watch running workflow
gh run watch
```

## Output

```markdown
## CI Status: feature/add-auth

| Workflow | Status | Duration | Commit | When |
|----------|--------|----------|--------|------|
| Tests | ✓ pass | 2m 34s | abc123 | 5m ago |
| Lint | ✓ pass | 45s | abc123 | 5m ago |
| Build | ✓ pass | 1m 12s | abc123 | 5m ago |

**All checks passing** ✓

---

Or if failing:

| Workflow | Status | Duration | Commit | When |
|----------|--------|----------|--------|------|
| Tests | ✗ fail | 1m 45s | abc123 | 5m ago |
| Lint | ✓ pass | 45s | abc123 | 5m ago |
| Build | - skip | - | abc123 | 5m ago |

**1 workflow failing**

### Tests Failure
```
--- FAIL: TestCreateUser
    expected 200, got 500
```

Run `/ci:fix` to analyse and fix.
```
