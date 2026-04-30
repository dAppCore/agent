---
name: release
description: Streamline the release process for modules
args: <patch|minor|major> [--preview]
---

# Release Workflow

This command automates the release process for modules. It handles version bumping, changelog generation, and Git tagging.

## Usage

```
/core:release patch           # Bump patch version
/core:release minor           # Bump minor version
/core:release major           # Bump major version
/core:release --preview       # Show what would happen
```

## Action

This command will execute the `release.sh` script:

```bash
"${CLAUDE_PLUGIN_ROOT}/scripts/release.sh" "<1>"
```
