---
name: check
description: Run QA checks without fixing (report only)
args: [--go|--php|--all]
---

# QA Check

Run QA pipeline and report issues without fixing them.

## Usage

```
/qa:check           # Auto-detect project type
/qa:check --go      # Force Go checks
/qa:check --php     # Force PHP checks
/qa:check --all     # Run both if applicable
```

## Process

1. **Detect project type**
2. **Run QA pipeline**
3. **Parse and report issues**
4. **Do NOT fix anything**

## Go Checks

```bash
core go qa
```

Runs:
- `go fmt` - Formatting
- `go vet` - Static analysis
- `golangci-lint` - Linting
- `go test` - Tests

## PHP Checks

```bash
core php qa
```

Runs:
- `pint` - Formatting
- `phpstan` - Static analysis
- `pest` - Tests

## Output

```markdown
## QA Report

**Project**: Go (go.mod detected)
**Status**: 3 issues found

### Formatting
✗ 2 files need formatting
- pkg/api/handler.go
- pkg/auth/token.go

### Linting
✗ 1 issue
- pkg/api/handler.go:42 - undefined: ErrNotFound

### Tests
✓ All passing (47/47)

---
**Summary**: fmt: FAIL | lint: FAIL | test: PASS

Run `/qa:qa` to fix these issues automatically.
```
