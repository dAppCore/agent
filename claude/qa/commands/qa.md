---
name: qa
description: Run full QA pipeline and fix all issues iteratively
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

Run the full QA pipeline and fix all issues until everything passes.

## Detection

First, detect the project type:
- If `go.mod` exists → Go project → `core go qa`
- If `composer.json` exists → PHP project → `core php qa`
- If both exist → check current directory or ask

## Process

1. **Run QA**: Execute `core go qa` or `core php qa`
2. **Parse issues**: Extract failures from output
3. **Fix each issue**: Address one at a time, simplest first
4. **Re-verify**: After fixes, re-run QA
5. **Repeat**: Until all checks pass
6. **Report**: Summary of what was fixed

## Issue Priority

Fix in this order (fastest feedback first):
1. **fmt** - formatting (auto-fix with `core go fmt`)
2. **lint** - static analysis (usually quick fixes)
3. **test** - failing tests (may need investigation)
4. **build** - compilation errors (fix before tests can run)

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

## Stop Condition

Only stop when:
- All QA checks pass, OR
- User explicitly cancels, OR
- Same error repeats 3 times (stuck - ask for help)
