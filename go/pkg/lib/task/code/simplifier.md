# Code Simplifier Task

Simplify recently modified code without changing behaviour.

## Process

1. Run `git diff --name-only origin/main..HEAD` to find changed files
2. Read each file, identify simplification opportunities
3. Apply changes one file at a time
4. `go build ./...` after each change to verify
5. If build breaks, revert

## Rules

- NEVER change public API
- NEVER change behaviour
- NEVER add features or comments
