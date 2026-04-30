---
name: qa
description: Run PHP QA checks and fix all issues iteratively
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

Run the full PHP QA pipeline and fix all issues.

## Process

1. **Run QA**: Execute `core php qa`
2. **Parse issues**: Extract failures from output (see format below)
3. **Fix each issue**: Address one at a time, simplest first
4. **Re-verify**: After fixes, re-run QA
5. **Repeat**: Until all checks pass
6. **Report**: Summary of what was fixed

## Issue Priority

Fix in this order (fastest feedback first):
1. **pint** - formatting issues (auto-fix with `core php fmt`)
2. **stan** - static analysis (usually quick fixes)
3. **test** - failing tests (may need more investigation)
4. **build** - runtime bootstrap or syntax errors (fix before tests can run)

## Output Parsing

### PHP QA Output
```
=== PINT ===
FAIL: 2 files need formatting

=== STAN ===
src/Http/Controller.php:42 - Undefined variable $user

=== TEST ===
✗ CreateUserTest::testSuccess
  Expected status 200, got 500

=== RESULT ===
pint: FAIL
stan: FAIL (1 error)
test: FAIL (1 failed)
```

## Fixing Strategy

**Formatting (pint):**
- Just run `core php fmt`
- No code reading needed

**Static analysis errors:**
- Read the specific file:line
- Understand the error type
- Make minimal fix

**Test failures:**
- Read the test file to understand expectation
- Read the implementation
- Fix the root cause (not just the symptom)

**Build errors:**
- Usually bootstrap, syntax, or autoload issues
- Fix before attempting other checks

## Stop Condition

Only stop when:
- All QA checks pass, OR
- User explicitly cancels, OR
- Same error repeats 3 times (stuck - ask for help)

## Example Session

```
Running: core php qa

## QA Issues

src/Http/Controller.php:42 - Undefined variable $user
✗ CreateUserTest::testSuccess

**Summary:** stan: FAIL (1) | test: FAIL (1)

---

Fixing static analysis issue: undefined variable
Reading src/Http/Controller.php...
Adding the missing variable assignment.

Running: core php qa

## QA Issues

✗ CreateUserTest::testSuccess
  Expected status 200, got 500

**Summary:** stan: PASS | test: FAIL (1)

---

Fixing test issue: expected 200, got 500
Reading test setup...
Correcting test data.

Running: core php qa

✓ All checks passed!

**Summary:**
- Fixed: undefined variable (added assignment)
- Fixed: TestCreateUser (corrected test setup)
- 2 issues resolved, all checks passing
```
