---
name: ci
description: Check CI status and manage workflows
args: [status|run|logs|fix]
---

# CI Integration

Check CI status and manage workflows using the `core` CLI (supports Forgejo and GitHub).

## Commands

### Status (default)
```
/ci:ci
/ci:ci status
```
```bash
core dev ci
core dev ci --branch $(git branch --show-current)
core dev ci --failed
```

### List workflows
```
/ci:ci workflows
```
```bash
core dev workflow list
```

### Issues
```
/ci:ci issues
```
```bash
core dev issues
core dev issues --assignee @me
```

### Reviews / PRs
```
/ci:ci reviews
```
```bash
core dev reviews
core dev reviews --all
```

### Fix failing CI
```
/ci:ci fix
```
Analyse failing CI and suggest fixes. See `/ci:fix`.
