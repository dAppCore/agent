---
name: workflow
description: Create or update CI workflow
args: <workflow-type>
---

# Workflow Generator

Create or update CI workflows. Forgejo Actions uses the same YAML format as GitHub Actions.

## Usage

```
/ci:workflow test
/ci:workflow lint
/ci:workflow release
```

## List existing workflows

```bash
core dev workflow list
```

## Sync workflows across repos

```bash
core dev workflow sync
```

## Workflow directory

Forgejo supports both:
- `.forgejo/workflows/` (preferred)
- `.github/workflows/` (also works)

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
          go-version-file: go.mod
      - run: go test -v -race ./...
```

### Go Release Workflow (core build)
```yaml
name: Release

on:
  push:
    tags: ['v*']

jobs:
  release:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
        with:
          fetch-depth: 0
      - uses: actions/setup-go@v5
        with:
          go-version-file: go.mod
      - name: Build and release
        run: core build release --we-are-go-for-launch
```

## Forgejo Notes

- Runner label: `ubuntu-latest` (maps to Forgejo runner labels)
- Secrets: Set via repo Settings → Actions → Secrets
- Runner: `build-noc` on the noc server
- Web UI: `forge.lthn.ai/{owner}/{repo}/actions`
