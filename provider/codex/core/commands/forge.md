---
name: forge
description: Work with Forge issues, pull requests, repositories, branch cleanup, and local repo sync
args: "[issue|pr|repo|branch] [subcommand] [options]"
---

# Forge Workflows

Use this family for Forge-backed issue, pull request, repository, and branch operations.

## Issues

```bash
core-agent issue list <repo> --org=core
core-agent issue get <repo> --number=N --org=core
core-agent issue create <repo> --title="..." --body="..." --labels="agentic,bug"
core-agent issue update <slug> --status=open --priority=high
core-agent issue assign <slug> --assignee=codex
core-agent issue comment <repo> --number=N --body="..."
core-agent issue report <slug> --report="..."
core-agent issue archive <slug>
```

## Pull Requests

```bash
core-agent pr list <repo> --org=core
core-agent pr get <repo> --number=N --org=core
core-agent pr merge <repo> --number=N --method=squash
core-agent pr close <repo> --number=N
```

## Repositories And Branches

```bash
core-agent repo list --org=core
core-agent repo get <repo> --org=core
core-agent repo sync <repo> --org=core --branch=dev
core-agent branch delete <repo> --branch=agent/fix-tests --org=core
```

For destructive branch operations, confirm the branch name and target repo explicitly before running the command.
