# feat(hooks): Define what to expose vs hide in output

## Summary

Document and implement a consistent policy for what hook output should be exposed to Claude vs hidden.

## Principles

**Expose (additionalContext):**
- Errors that need fixing
- Failures that block progress
- Decisions that need to be made
- Security warnings
- Breaking changes

**Hide (suppressOutput):**
- Success confirmations ("formatted", "passed")
- Verbose progress output
- Repetitive status messages
- Debug information
- Intermediate results

## Output Categories

### Always Expose

| Category | Example | Reason |
|----------|---------|--------|
| Test failures | `FAIL: TestFoo` | Must be fixed |
| Build errors | `cannot find package` | Blocks progress |
| Lint errors | `undefined: foo` | Code quality |
| Security alerts | `HIGH vulnerability` | Critical |
| Type errors | `type mismatch` | Must be fixed |

### Always Hide

| Category | Example | Reason |
|----------|---------|--------|
| Pass confirmations | `PASS: TestFoo` | No action needed |
| Format success | `Formatted 3 files` | No action needed |
| Coverage numbers | `coverage: 84.2%` | Unless requested |
| Timing info | `(12.3s)` | Noise |
| Progress bars | `[=====>   ]` | Noise |

### Conditional

| Category | Show When | Hide When |
|----------|-----------|-----------|
| Warnings | First occurrence | Repeated |
| Suggestions | Actionable | Informational |
| Diffs | Small (<10 lines) | Large |
| Stack traces | Unique error | Repeated |

## Implementation Pattern

```bash
#!/bin/bash
INPUT=$(cat)
OUTPUT=$(echo "$INPUT" | jq -r '.tool_response.output // empty')

# Extract what matters
ERRORS=$(echo "$OUTPUT" | grep -E "FAIL|ERROR|panic" | head -10)
SUMMARY=$(echo "$OUTPUT" | tail -1)

if [ -n "$ERRORS" ]; then
  # Expose failures
  cat << EOF
{
  "hookSpecificOutput": {
    "hookEventName": "PostToolUse",
    "additionalContext": "## Issues Found\n\n$ERRORS\n\n**Summary:** $SUMMARY"
  }
}
EOF
else
  # Hide success
  echo '{"suppressOutput": true}'
fi
```

## Hook-Specific Policies

### PostToolUse (Bash)

```
core go test  → Expose failures, hide passes
core go lint  → Expose errors, hide "no issues"
core build    → Expose errors, hide success
git status    → Always expose (informational)
git commit    → Expose result, hide diff
```

### PostToolUse (Write/Edit)

```
Format result → Always hide
Debug check   → Expose if found, hide if clean
```

### PostToolUseFailure

```
All errors    → Always expose with context
Interrupts    → Hide (user initiated)
```

## Aggregation

When multiple issues, aggregate intelligently:

```
Instead of:
- FAIL: TestA
- FAIL: TestB
- FAIL: TestC
- (47 more)

Show:
"50 tests failed. Top failures:
- TestA: nil pointer
- TestB: timeout
- TestC: assertion failed
Run `core go test -v` for full output"
```
