# feat(hooks): Auto-run tests after code changes (async)

## Summary

Add async PostToolUse hooks that automatically run tests after Write/Edit operations, reporting results without blocking.

## Problem

Claude often edits code but forgets to run tests, or runs tests manually each time which is slow and repetitive.

## Solution

Async PostToolUse hooks that:
1. Detect code file changes (*.go, *.php, *.ts)
2. Run relevant tests in background
3. Report results on next turn via `systemMessage`

## Hook Configuration

```json
{
  "hooks": {
    "PostToolUse": [
      {
        "matcher": "Write|Edit",
        "hooks": [
          {
            "type": "command",
            "command": "core ai auto-test",
            "async": true,
            "timeout": 120,
            "statusMessage": "Running tests..."
          }
        ]
      }
    ]
  }
}
```

## Test Detection Logic

`core ai auto-test` should:

1. **Read file path from stdin:**
   ```bash
   FILE=$(jq -r '.tool_input.file_path')
   ```

2. **Detect test scope:**
   - `*.go` in `pkg/foo/` → `core go test ./pkg/foo/...`
   - `*.php` in `src/` → `core php test --filter=related`
   - `*_test.go` → run that specific test file
   - `*.spec.ts` → run that specific spec

3. **Run minimal test set:**
   - Don't run full suite on every change
   - Use `--filter` or package scope
   - Cache recent test runs

4. **Report results:**
   ```json
   {
     "systemMessage": "Tests for pkg/foo: 12 passed, 0 failed (2.3s)"
   }
   ```

## Smart Test Selection

For Go:
```bash
# Find related test file
TEST_FILE="${FILE%.go}_test.go"
if [ -f "$TEST_FILE" ]; then
  core go test -run "$(basename $TEST_FILE .go)" ./...
else
  # Run package tests
  PKG_DIR=$(dirname "$FILE")
  core go test "./$PKG_DIR/..."
fi
```

For PHP:
```bash
# Find related test
TEST_FILE=$(echo "$FILE" | sed 's/\.php$/Test.php/' | sed 's|src/|tests/|')
if [ -f "$TEST_FILE" ]; then
  core php test "$TEST_FILE"
fi
```

## Debouncing

Avoid running tests multiple times for rapid edits:
- Track last test run per package
- Skip if <5 seconds since last run
- Queue and batch rapid changes
