# Test Coverage Gaps

Find untested code paths and write tests for them.

## Process

1. `go test -cover ./...` to identify coverage
2. Focus on exported functions with 0% coverage
3. Write tests using Good/Bad/Ugly naming
4. Prioritise: error paths > happy paths > edge cases
