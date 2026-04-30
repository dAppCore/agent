---
name: release
description: Release a new version of a module
args: [patch|minor|major]
flags:
  preview:
    description: Show what would happen without actually making a release
    type: boolean
    default: false
---

# Release new version

Streamlines the release process for modules.

## Commands

### Bump patch version
`/core:release patch`

### Bump minor version
`/core:release minor`

### Bump major version
`/core:release major`

### Preview release
`/core:release patch --preview`

## Workflow

1.  **Bump version**: Bumps the version in `package.json` and other necessary files.
2.  **Update CHANGELOG.md**: Generates a new entry in the changelog based on commit history.
3.  **Create git tag**: Creates a new git tag for the release.
4.  **Push tag**: Pushes the new tag to the remote repository.
5.  **Trigger CI release**: The new tag should trigger the CI/CD release pipeline.

## Implementation

This command is implemented by the `release.sh` script.

```bash
/bin/bash ../scripts/release.sh "$@"
```
