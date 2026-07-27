<!-- SPDX-License-Identifier: EUPL-1.2 -->
# Formatting, linting & coding standards

## Formatting & linting

### Go

```bash
cd go
gofmt -w .
golangci-lint run --timeout=5m --tests=false ./...
go vet ./...
```

### PHP

```bash
composer lint                 # Laravel Pint, PSR-12
./vendor/bin/pint --dirty     # only changed files
```

### Automatic formatting

The `core` plugin's PostToolUse hooks (`provider/claude/core/scripts/`) auto-format after
every edit: `go-format.sh` (gofmt on edited `.go`), `php-format.sh` (pint on edited `.php`),
and `check-debug.sh` (warns about `dd()`, `dump()`, `fmt.Println()` left in code).

## Coding standards

### Language

Use **UK English** throughout: colour, organisation, centre, licence, behaviour,
catalogue. Never American spellings.

### Go

- standard `gofmt` formatting
- errors via `core.E("scope.Method", "what failed", err)` where the core framework is used
- exported types get doc comments
- test files co-locate with their source

### PHP

- `declare(strict_types=1);` in every file
- all parameters and return types type-hinted
- PSR-12 via Laravel Pint
- Pest syntax for tests (not PHPUnit)

### Shell scripts

- shebang `#!/bin/bash`
- read JSON input with `jq`
- hook output: JSON with `decision` + optional `message`

### Commits

Conventional commits — `type(scope): description`:

```
feat(lifecycle): add exponential backoff to dispatcher
fix(brain): handle empty embedding vectors
docs(architecture): update data flow diagram
```
