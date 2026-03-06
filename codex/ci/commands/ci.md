---
name: ci
description: Check CI status and manage workflows
args: [status|run|logs|fix]
---

# CI Integration

Check GitHub Actions status and manage CI workflows.

## Commands

### Status (default)
```
/ci:ci
/ci:ci status
```

Check current CI status for the repo/branch.

### Run workflow
```
/ci:ci run
/ci:ci run tests
```

Trigger a workflow run.

### View logs
```
/ci:ci logs
/ci:ci logs 12345
```

View logs from a workflow run.

### Fix failing CI
```
/ci:ci fix
```

Analyse failing CI and suggest fixes.

## Implementation

### Check status
```bash
gh run list --limit 5
gh run view --log-failed
```

### Trigger workflow
```bash
gh workflow run tests.yml
```

### View logs
```bash
gh run view 12345 --log
```

## CI Status Report

```markdown
## CI Status: main

| Workflow | Status | Duration | Commit |
|----------|--------|----------|--------|
| Tests | ✓ passing | 2m 34s | abc123 |
| Lint | ✓ passing | 45s | abc123 |
| Build | ✗ failed | 1m 12s | abc123 |

### Failing: Build
```
Error: go build failed
  pkg/api/handler.go:42: undefined: ErrNotFound
```

**Suggested fix**: Add missing error definition
```
