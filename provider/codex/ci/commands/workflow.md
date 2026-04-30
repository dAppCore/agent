---
name: workflow
description: Create or update GitHub Actions workflow
args: <workflow-type>
---

# Workflow Generator

Create or update GitHub Actions workflows.

## Workflow Types

### test
Standard test workflow for Go/PHP projects.

### lint
Linting workflow with golangci-lint or PHPStan.

### release
Release workflow with goreleaser or similar.

### deploy
Deployment workflow (requires configuration).

## Usage

```
/ci:workflow test
/ci:workflow lint
/ci:workflow release
```

## Templates

### Go Test Workflow
```yaml
name: Tests

on:
  push:
    branches: [main]
  pull_request:
    branches: [main]

jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with:
          go-version: '1.22'
      - run: go test -v ./...
```

### PHP Test Workflow
```yaml
name: Tests

on:
  push:
    branches: [main]
  pull_request:
    branches: [main]

jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: shivammathur/setup-php@v2
        with:
          php-version: '8.3'
      - run: composer install
      - run: composer test
```
