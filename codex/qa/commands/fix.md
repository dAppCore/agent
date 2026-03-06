---
name: fix
description: Fix a specific QA issue
args: <issue-description>
---

# Fix Issue

Fix a specific issue from QA output.

## Usage

```
/qa:fix undefined: ErrNotFound in pkg/api/handler.go:42
/qa:fix TestCreateUser failing - expected 200, got 500
/qa:fix pkg/api/handler.go needs formatting
```

## Process

1. **Parse the issue**: Extract file, line, error type
2. **Read context**: Read the file around the error line
3. **Understand**: Determine root cause
4. **Fix**: Make minimal change to resolve
5. **Verify**: Run relevant test/lint check

## Issue Types

### Undefined variable/type
```
undefined: ErrNotFound
```
→ Add missing import or define the variable

### Test failure
```
expected 200, got 500
```
→ Read test and implementation, fix logic

### Formatting
```
file needs formatting
```
→ Run `core go fmt` or `core php fmt`

### Lint warning
```
ineffectual assignment to err
```
→ Use the variable or remove assignment

### Type error
```
cannot use X as Y
```
→ Fix type conversion or function signature
