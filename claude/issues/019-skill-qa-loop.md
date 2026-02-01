# feat(skill): Add /core:qa iterative QA fix loop

## Summary

Add a `/core:qa` skill that runs the full QA pipeline, extracts actionable issues, and guides Claude through fixing them one by one until everything passes.

## Use Case

```
/core:qa
```

Runs `core go qa` or `core php qa`, parses output, and iteratively fixes issues:

```
[QA] Running core go qa...
[QA] Found 3 issues:

1. ✗ lint: undefined: ErrNotFound (pkg/api/handler.go:42)
2. ✗ test: TestCreateUser failed - expected 200, got 500
3. ✗ fmt: pkg/api/handler.go needs formatting

Fixing issue 1/3: undefined: ErrNotFound
→ Reading pkg/api/handler.go...
→ Adding missing import...
→ Fixed.

Fixing issue 2/3: TestCreateUser failed
→ Reading test output...
→ The handler returns 500 because ErrNotFound wasn't defined
→ Already fixed by issue 1, re-running test...
→ Passed.

Fixing issue 3/3: formatting
→ Running core go fmt...
→ Fixed.

[QA] Re-running full QA...
[QA] ✓ All checks passed!
```

## Skill Definition

`claude/commands/qa.md`:

```markdown
---
name: qa
description: Run QA checks and fix all issues iteratively
hooks:
  PostToolUse:
    - matcher: "Bash"
      hooks:
        - type: command
          command: "core ai qa-filter"
  Stop:
    - hooks:
        - type: command
          command: "core ai qa-verify"
          once: true
---

# QA Fix Loop

Run the full QA pipeline and fix all issues.

## Detection

First, detect the project type:
- If `go.mod` exists → Go project → `core go qa`
- If `composer.json` exists → PHP project → `core php qa`
- If both exist → ask user or check current directory

## Process

1. **Run QA**: Execute `core go qa` or `core php qa`
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

**Formatting (fmt/pint):**
- Just run `core go fmt` or `core php fmt`
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
- Same error repeats 3 times (stuck)
```

## Hook Implementations

### core ai qa-filter

Filters QA output to show only actionable issues:

```bash
#!/bin/bash
INPUT=$(cat)
COMMAND=$(echo "$INPUT" | jq -r '.tool_input.command // empty')
OUTPUT=$(echo "$INPUT" | jq -r '.tool_response.output // empty')

# Only process QA commands
case "$COMMAND" in
  "core go qa"*|"core php qa"*|"core go test"*|"core php test"*|"core go lint"*|"core php stan"*)
    ;;
  *)
    echo "$INPUT"
    exit 0
    ;;
esac

# Extract failures only
FAILURES=$(echo "$OUTPUT" | grep -E "^(FAIL|---\s*FAIL|✗|ERROR|undefined:|error:)" | head -20)
SUMMARY=$(echo "$OUTPUT" | grep -E "^(fmt:|lint:|test:|pint:|stan:|RESULT)" | tail -5)

if [ -z "$FAILURES" ]; then
  # All passed - show brief summary
  echo '{"suppressOutput": true, "hookSpecificOutput": {"hookEventName": "PostToolUse", "additionalContext": "✓ QA passed"}}'
else
  # Show only failures
  cat << EOF
{
  "suppressOutput": true,
  "hookSpecificOutput": {
    "hookEventName": "PostToolUse",
    "additionalContext": "## QA Issues\n\n\`\`\`\n$FAILURES\n\`\`\`\n\n**Summary:** $SUMMARY"
  }
}
EOF
fi
```

### core ai qa-verify

Stop hook that ensures QA passes before stopping:

```bash
#!/bin/bash
INPUT=$(cat)
STOP_ACTIVE=$(echo "$INPUT" | jq -r '.stop_hook_active')

if [ "$STOP_ACTIVE" = "true" ]; then
  exit 0
fi

# Detect project type and run QA
if [ -f "go.mod" ]; then
  RESULT=$(core go qa 2>&1)
elif [ -f "composer.json" ]; then
  RESULT=$(core php qa 2>&1)
else
  exit 0
fi

# Check if passed
if echo "$RESULT" | grep -q "FAIL\|ERROR\|✗"; then
  ISSUES=$(echo "$RESULT" | grep -E "^(FAIL|ERROR|✗|undefined:)" | head -5)
  cat << EOF
{
  "decision": "block",
  "reason": "QA still has issues:\n$ISSUES\n\nPlease fix these before stopping."
}
EOF
else
  exit 0
fi
```

## Example Session

```
> /core:qa

Detecting project type... Found go.mod → Go project

Running: core go qa

## QA Issues

```
pkg/api/handler.go:42:15: undefined: ErrNotFound
--- FAIL: TestCreateUser (0.02s)
```

**Summary:** lint: FAIL (1) | test: FAIL (1)

---

Fixing lint issue: undefined ErrNotFound

Reading pkg/api/handler.go...
The error variable is used but not defined. Adding:

var ErrNotFound = errors.New("not found")

Running: core go qa

## QA Issues

```
--- FAIL: TestCreateUser (0.02s)
    expected 200, got 404
```

**Summary:** lint: PASS | test: FAIL (1)

---

Fixing test issue: expected 200, got 404

Reading the test... it expects success but handler now returns 404.
The test data doesn't have a matching record. Fixing test setup...

Running: core go qa

✓ QA passed

---

**Summary:**
- Fixed: undefined ErrNotFound (added error variable)
- Fixed: TestCreateUser (corrected test setup)
- 2 issues resolved, all checks passing
```

## Future CLI Integration

```bash
core ai qa              # Run QA fix loop
core ai qa --check-only # Just report, don't fix
core ai qa --fmt-only   # Only fix formatting
```
