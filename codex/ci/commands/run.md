---
name: run
description: Trigger a CI workflow run
args: [workflow-name]
---

# Run Workflow

Manually trigger a GitHub Actions workflow.

## Usage

```
/ci:run              # Run default workflow
/ci:run tests        # Run specific workflow
/ci:run release      # Trigger release workflow
```

## Process

1. **List available workflows**
   ```bash
   gh workflow list
   ```

2. **Trigger workflow**
   ```bash
   gh workflow run tests.yml
   gh workflow run tests.yml --ref feature-branch
   ```

3. **Watch progress**
   ```bash
   gh run watch
   ```

## Common Workflows

| Workflow | Trigger | Purpose |
|----------|---------|---------|
| `tests.yml` | Push, PR | Run test suite |
| `lint.yml` | Push, PR | Run linters |
| `build.yml` | Push | Build artifacts |
| `release.yml` | Tag | Create release |
| `deploy.yml` | Manual | Deploy to environment |

## Output

```markdown
## Workflow Triggered

**Workflow**: tests.yml
**Branch**: feature/add-auth
**Run ID**: 12345

Watching progress...

```
⠋ Tests running...
  ✓ Setup (12s)
  ✓ Install dependencies (45s)
  ⠋ Run tests (running)
```

**Run completed in 2m 34s** ✓
```

## Options

```bash
# Run with inputs (for workflows that accept them)
gh workflow run deploy.yml -f environment=staging

# Run on specific ref
gh workflow run tests.yml --ref main
```
