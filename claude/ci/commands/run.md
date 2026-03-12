---
name: run
description: Trigger a CI workflow run
args: [workflow-name]
---

# Run Workflow

Trigger a CI workflow or view available workflows.

## Usage

```
/ci:run              # List available workflows
/ci:run tests        # Trigger specific workflow
```

## Commands

```bash
# List available workflows
core dev workflow list

# Sync workflows across repos
core dev workflow sync
```

## Notes

- Forgejo Actions uses `.forgejo/workflows/` or `.github/workflows/` (both supported)
- Workflows are triggered automatically on push/PR/tag
- Manual dispatch: use the Forgejo web UI at `forge.lthn.ai/{owner}/{repo}/actions`
- Runner: `build-noc` on the noc server
