---
name: verify
description: Verify work is complete before stopping
args: [--quick|--full]
---

# Work Verification

Verify that PHP work is complete and ready to commit/push.

## Arguments

- No args: Standard verification
- `--quick`: Fast checks only (format, lint)
- `--full`: All checks including slow tests

## Verification Steps

### 1. Check for uncommitted changes
```bash
git status --porcelain
```

### 2. Check for debug statements
Look for:
- Go: `fmt.Println`, `log.Println`, `spew.Dump`
- PHP: `dd(`, `dump(`, `var_dump(`, `ray(`
- JS/TS: `console.log`, `debugger`

### 3. Run tests
```bash
core php test
```

### 4. Run linter
```bash
core php stan
```

### 5. Check formatting
```bash
core php fmt --test
```

## Output

Report verification status:

```
## Verification Results

✓ No uncommitted changes
✓ No debug statements found
✓ Tests passing (47/47)
✓ Lint clean
✓ Formatting correct

**Status: READY**
```

Or if issues found:

```
## Verification Results

✓ No uncommitted changes
✗ Debug statement found: src/handler.go:42
✗ Tests failing (45/47)
✓ Lint clean
✓ Formatting correct

**Status: NOT READY**

Fix these issues before proceeding.
```
