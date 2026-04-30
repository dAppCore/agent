# Hook Output Policy

Consistent policy for what hook output to expose to Claude vs hide.

## Principles

### Always Expose

| Category | Example | Reason |
|----------|---------|--------|
| Test failures | `FAIL: TestFoo` | Must be fixed |
| Build errors | `cannot find package` | Blocks progress |
| Lint errors | `undefined: foo` | Code quality |
| Security alerts | `HIGH vulnerability` | Critical |
| Type errors | `type mismatch` | Must be fixed |
| Debug statements | `dd() found` | Must be removed |
| Uncommitted work | `3 files unstaged` | Might get lost |
| Coverage drops | `84% → 79%` | Quality regression |

### Always Hide

| Category | Example | Reason |
|----------|---------|--------|
| Pass confirmations | `PASS: TestFoo` | No action needed |
| Format success | `Formatted 3 files` | No action needed |
| Coverage stable | `84% (unchanged)` | No action needed |
| Timing info | `(12.3s)` | Noise |
| Progress bars | `[=====>   ]` | Noise |

### Conditional

| Category | Show When | Hide When |
|----------|-----------|-----------|
| Warnings | First occurrence | Repeated |
| Suggestions | Actionable | Informational |
| Diffs | Small (<10 lines) | Large |
| Stack traces | Unique error | Repeated |

## Implementation

Use `output-policy.sh` helper functions:

```bash
source "$SCRIPT_DIR/output-policy.sh"

# Expose failures
expose_error "Build failed" "$error_details"
expose_warning "Debug statements found" "$locations"

# Hide success
hide_success

# Pass through unchanged
pass_through "$input"
```

## Hook-Specific Policies

| Hook | Expose | Hide |
|------|--------|------|
| `check-debug.sh` | Debug statements found | Clean file |
| `post-commit-check.sh` | Uncommitted work | Clean working tree |
| `check-coverage.sh` | Coverage dropped | Coverage stable/improved |
| `go-format.sh` | (never) | Always silent |
| `php-format.sh` | (never) | Always silent |

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
- TestC: assertion failed"
```
