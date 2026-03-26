---
name: pr-merge
description: This skill should be used when the user asks to "merge PR", "merge pull request", "accept PR", or needs to merge a Forge PR. Supports merge, rebase, and squash methods.
argument-hint: <repo> --number=N [--method=merge|rebase|squash] [--org=core]
allowed-tools: ["Bash"]
---

# Merge Forge Pull Request

Merge a PR on Forge. Default method is merge.

```bash
core-agent pr/merge <repo> --number=N [--method=merge|rebase|squash] [--org=core]
```

Example:
```bash
core-agent pr/merge go --number=22
core-agent pr/merge go-forge --number=7 --method=squash
```

## Important

- Always confirm with the user before merging
- Check PR status with `pr/get` first if unsure about mergeability
- The merge happens on Forge, not locally
