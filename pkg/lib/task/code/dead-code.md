# Dead Code Scan

Find and remove unreachable code, unused functions, and orphaned files.

## Process

1. `go vet ./...` for compiler-detected dead code
2. Search for unexported functions with zero callers
3. Check for unreachable branches (always-true conditions)
4. Verify before deleting — some code is used via reflection or build tags
