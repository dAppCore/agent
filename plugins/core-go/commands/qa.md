---
name: qa
description: Run Go QA checks and fix all issues iteratively
hooks:
  PostToolUse:
    - matcher: "Bash"
      hooks:
        - type: command
          command: "${CLAUDE_PLUGIN_ROOT}/scripts/qa-filter.sh"
  Stop:
    - hooks:
        - type: command
          command: "${CLAUDE_PLUGIN_ROOT}/scripts/qa-verify.sh"
          once: true
---

# QA Fix Loop

Run the full Go QA pipeline and fix all issues.

## Process

1. **Run QA**: Execute `core go qa`
2. **Parse issues**: Extract failures from output (see format below)
3. **Fix each issue**: Address one at a time, simplest first
4. **Re-verify**: After fixes, re-run QA
5. **Repeat**: Until all checks pass
6. **Report**: Summary of what was fixed

## Issue Priority

Fix in this order (fastest feedback first):
1. **fmt** - formatting issues (auto-fix with `core go fmt`)
2. **lint** - static analysis (usually quick fixes)
3. **test** - failing tests (may need more investigation)
4. **build** - compilation errors (fix before tests can run)

## Output Parsing

### Go QA Output
```
=== FMT ===
FAIL: pkg/api/handler.go needs formatting

=== LINT ===
pkg/api/handler.go:42:15: undefined: ErrNotFound (typecheck)
pkg/api/handler.go:87:2: ineffectual assignment to err (ineffassign)

=== TEST ===
--- FAIL: TestCreateUser (0.02s)
    handler_test.go:45: expected 200, got 500
FAIL

=== RESULT ===
fmt: FAIL
lint: FAIL (2 issues)
test: FAIL (1 failed)
```

## Fixing Strategy

**Formatting (fmt):**
- Just run `core go fmt`
- No code reading needed

**Lint errors:**
- Read the specific file:line
- Understand the error type
- Make minimal fix

**Test failures:**
- Read the test file to understand expectation
- Read the implementation
- Fix the root cause (not just the symptom)

**Build errors:**
- Usually missing imports or typos
- Fix before attempting other checks

## Stop Condition

Only stop when:
- All QA checks pass, OR
- User explicitly cancels, OR
- Same error repeats 3 times (stuck - ask for help)

## Example Session

```
Running: core go qa

## QA Issues

pkg/api/handler.go:42:15: undefined: ErrNotFound
--- FAIL: TestCreateUser (0.02s)

**Summary:** lint: FAIL (1) | test: FAIL (1)

---

Fixing lint issue: undefined ErrNotFound
Reading pkg/api/handler.go...
Adding error variable definition.

Running: core go qa

## QA Issues

--- FAIL: TestCreateUser (0.02s)
    expected 200, got 404

**Summary:** lint: PASS | test: FAIL (1)

---

Fixing test issue: expected 200, got 404
Reading test setup...
Correcting test data.

Running: core go qa

✓ All checks passed!

**Summary:**
- Fixed: undefined ErrNotFound (added error variable)
- Fixed: TestCreateUser (corrected test setup)
- 2 issues resolved, all checks passing
```
