---
name: fix
description: Analyse and fix failing CI
---

# Fix CI

Analyse failing CI runs and suggest/apply fixes.

## Process

1. **Get failing run**
   ```bash
   gh run list --status failure --limit 1
   gh run view <id> --log-failed
   ```

2. **Analyse failure**
   - Parse error messages
   - Identify root cause
   - Check if local issue or CI-specific

3. **Suggest fix**
   - Code changes if needed
   - CI config changes if needed

4. **Apply fix** (if approved)

## Common CI Failures

### Test Failures
```
Error: go test failed
--- FAIL: TestFoo
```
→ Fix the failing test locally, then push

### Lint Failures
```
Error: golangci-lint failed
file.go:42: undefined: X
```
→ Fix lint issue locally

### Build Failures
```
Error: go build failed
cannot find package
```
→ Run `go mod tidy`, check imports

### Dependency Issues
```
Error: go mod download failed
```
→ Check go.mod, clear cache, retry

### Timeout
```
Error: Job exceeded time limit
```
→ Optimise tests or increase timeout in workflow

## Output

```markdown
## CI Failure Analysis

**Run**: #12345
**Workflow**: Tests
**Failed at**: 2024-01-15 14:30

### Error
```
--- FAIL: TestCreateUser (0.02s)
    handler_test.go:45: expected 200, got 500
```

### Analysis
The test expects a 200 response but gets 500. This indicates the handler is returning an error.

### Root Cause
Looking at recent changes, `ErrNotFound` was removed but still referenced.

### Fix
Add the missing error definition:
```go
var ErrNotFound = errors.New("not found")
```

### Commands
```bash
# Apply fix and push
git add . && git commit -m "fix: add missing ErrNotFound"
git push
```
```
