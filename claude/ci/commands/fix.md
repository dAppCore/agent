---
name: fix
description: Analyse and fix failing CI
---

# Fix CI

Analyse failing CI runs and suggest/apply fixes.

## Process

1. **Get failing runs**
   ```bash
   core dev ci --failed
   ```

2. **Analyse failure**
   - Parse error messages from CI output
   - Identify root cause
   - Check if local issue or CI-specific

3. **Reproduce locally**
   ```bash
   core go test
   core go lint
   core go vet
   ```

4. **Suggest fix**
   - Code changes if needed
   - CI config changes if needed

5. **Apply fix** (if approved)

## Common CI Failures

### Test Failures
→ Run `core go test --run TestFoo`, fix the test, push

### Lint Failures
→ Run `core go lint`, fix lint issues

### Build Failures
→ Run `core build`, check imports, run `core go fmt`

### Dependency Issues
→ Check go.mod, `core go fmt`, retry
