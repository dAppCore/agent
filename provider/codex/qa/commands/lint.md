---
name: lint
description: Run linter and fix issues
args: [--check|--fix]
---

# Lint

Run linter and optionally fix issues.

## Usage

```
/qa:lint            # Run lint, report issues
/qa:lint --check    # Check only, no fixes
/qa:lint --fix      # Auto-fix where possible
```

## Process

### Go
```bash
# Check
core go lint

# Some issues can be auto-fixed
golangci-lint run --fix
```

### PHP
```bash
# Check
core php stan

# PHPStan doesn't auto-fix, but can suggest fixes
```

## Common Issues

### Go

| Issue | Fix |
|-------|-----|
| `undefined: X` | Add import or define variable |
| `ineffectual assignment` | Use variable or remove |
| `unused parameter` | Use `_` prefix or remove |
| `error return value not checked` | Handle the error |

### PHP

| Issue | Fix |
|-------|-----|
| `Undefined variable` | Define or check existence |
| `Parameter $x has no type` | Add type hint |
| `Method has no return type` | Add return type |

## Output

```markdown
## Lint Results

**Linter**: golangci-lint
**Issues**: 3

### Errors
1. **pkg/api/handler.go:42** - undefined: ErrNotFound
   → Add `var ErrNotFound = errors.New("not found")`

2. **pkg/api/handler.go:87** - error return value not checked
   → Handle error: `if err != nil { return err }`

### Warnings
1. **pkg/api/handler.go:15** - unused parameter ctx
   → Rename to `_` or use it

---
Run `/qa:lint --fix` to auto-fix where possible.
```
