---
name: status
description: Show CI status for current branch
---

# CI Status

Show CI status for the current branch.

## Usage

```
/ci:status
```

## Detection

Detect the CI provider from git remote, then use the appropriate method:

```bash
REMOTE_URL=$(git remote get-url origin 2>/dev/null)
```

### Forgejo (forge.lthn.ai)

```bash
# Extract owner/repo from remote
OWNER_REPO=$(git remote get-url origin 2>/dev/null | sed -E 's#.*forge\.lthn\.ai[:/]+([0-9]+/)?##; s#\.git$##')

# List recent workflow runs (requires FORGEJO_TOKEN or use web UI)
curl -s "https://forge.lthn.ai/api/v1/repos/${OWNER_REPO}/actions/tasks?limit=10&state=running"

# Or just open the Actions page
echo "https://forge.lthn.ai/${OWNER_REPO}/actions"
```

### GitHub (fallback, requires gh CLI)

```bash
core dev ci
core dev ci --branch $(git branch --show-current)
core dev ci --failed
```

## Output

Present results as a status table:

```markdown
## CI Status: main

| Workflow | Status | When |
|----------|--------|------|
| Tests | pass | 5m ago |
| Build | pass | 5m ago |

**All checks passing**
```

If no API token available, output the web URL:
```
View CI status: https://forge.lthn.ai/{owner}/{repo}/actions
```
