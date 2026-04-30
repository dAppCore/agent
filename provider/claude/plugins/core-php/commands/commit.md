---
name: commit
description: Generate a conventional commit message for staged changes
args: "[message]"
flags:
  - --amend
hooks:
  Before:
    - hooks:
        - type: command
          command: "${CLAUDE_PLUGIN_ROOT}/scripts/smart-commit.sh"
---

# Smart Commit

Generate a conventional commit message for staged changes.

## Usage

Generate message automatically:
`/core-php:commit`

Provide a custom message:
`/core-php:commit "feat(module): add action validation"`

Amend the previous commit:
`/core-php:commit --amend`

## Behavior

1.  **Analyze Staged Changes**: Examines the `git diff --staged` to understand the nature of the changes.
2.  **Generate Conventional Commit Message**:
    -   `feat`: For new files, functions, or features.
    -   `fix`: For bug fixes.
    -   `refactor`: For code restructuring without changing external behavior.
    -   `docs`: For changes to documentation.
    -   `test`: For adding or modifying tests.
    -   `chore`: For routine maintenance tasks.
3.  **Determine Scope**: Infers the scope from the affected module's file paths (e.g., `auth`, `payment`, `ui`).
4.  **Add Co-Authored-By Trailer**: Appends `Co-Authored-By: Claude <noreply@anthropic.com>` to the commit message.

## Message Generation Example

```
feat(auth): add JWT token validation

- Add validateToken() function
- Add token expiry check
- Add unit tests for validation

Co-Authored-By: Claude <noreply@anthropic.com>
```
