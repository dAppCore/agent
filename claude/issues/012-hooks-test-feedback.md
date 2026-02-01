# feat(hooks): Reduce test noise with smart output filtering

## Summary

Add PostToolUse hooks that filter test output to show only what matters - failures and summaries, not passing tests.

## Problem

When running `core go test` or `core php test`, the full output floods the context:
- Hundreds of "PASS" lines for passing tests
- Verbose coverage output
- Repetitive timing information

This wastes context window and makes failures harder to spot.

## Solution

PostToolUse hooks that:
1. Parse test output
2. Extract only failures, errors, and summary
3. Return filtered output as `additionalContext`
4. Use `suppressOutput: true` to hide the verbose original

## Hook Configuration

```json
{
  "hooks": {
    "PostToolUse": [
      {
        "matcher": "Bash",
        "hooks": [
          {
            "type": "command",
            "command": "core ai filter test-output"
          }
        ]
      }
    ]
  }
}
```

## Filter Logic

**Go tests:**
- Show: FAIL, ERROR, panic, --- FAIL
- Hide: PASS, RUN, coverage lines (unless requested)
- Summary: "47 passed, 2 failed, 1 skipped"

**PHP/Pest tests:**
- Show: ✗, FAIL, Error, Exception
- Hide: ✓, passing test names
- Summary: "Tests: 47 passed, 2 failed"

**Lint output:**
- Show: actual errors/warnings with file:line
- Hide: "no issues found" for each file
- Summary: "3 issues in 2 files"

## Output Format

```json
{
  "suppressOutput": true,
  "hookSpecificOutput": {
    "hookEventName": "PostToolUse",
    "additionalContext": "## Test Results\n\n✗ 2 failed, 47 passed\n\n### Failures:\n- TestFoo: expected 5, got 3\n- TestBar: nil pointer"
  }
}
```

## Command Detection

Only filter when command matches:
- `core go test*`
- `core php test*`
- `core go lint*`
- `core php stan*`
- `core qa*`
