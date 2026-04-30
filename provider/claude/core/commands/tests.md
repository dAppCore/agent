---
name: tests
description: Verify tests pass for changed files
---

# Test Verification

Run tests related to changed files.

## Process

1. **Identify changed files**
   ```bash
   git diff --name-only HEAD
   ```

2. **Find related tests**
   - Go: `*_test.go` files in same package
   - PHP: `*Test.php` files in tests/ directory

3. **Run targeted tests**
   ```bash
   # Go - run package tests
   core go test ./pkg/changed/...

   # PHP - run filtered tests
   core php test --filter=ChangedTest
   ```

4. **Report results**

## Smart Test Detection

### Go
```
Changed: pkg/api/handler.go
Related: pkg/api/handler_test.go
Run: core go test ./pkg/api/...
```

### PHP
```
Changed: src/Http/UserController.php
Related: tests/Http/UserControllerTest.php
Run: core php test tests/Http/UserControllerTest.php
```

## Output

```
## Test Verification

**Changed files**: 3
**Related tests**: 2 packages

### Results
✓ pkg/api: 12 tests passed
✓ pkg/auth: 8 tests passed

**All tests passing!**
```

Or:

```
## Test Verification

**Changed files**: 3
**Related tests**: 2 packages

### Results
✓ pkg/api: 12 tests passed
✗ pkg/auth: 1 failed

### Failures
- TestValidateToken: expected true, got false
  auth_test.go:45

**Fix failing tests before committing.**
```
