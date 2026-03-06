#!/bin/bash
# This script validates a git branch name based on a naming convention.

full_command="${CLAUDE_TOOL_INPUT:-$*}"

# Check for override flag
if [[ "$full_command" =~ --no-verify ]]; then
    echo "✓ Branch validation skipped due to --no-verify flag."
    exit 0
fi

branch_name=""

# Regex to find branch name from 'git checkout -b <branch> ...'
if [[ "$full_command" =~ git\ checkout\ -b\ ([^[:space:]]+) ]]; then
    branch_name="${BASH_REMATCH[1]}"
# Regex to find branch name from 'git branch <branch> ...'
elif [[ "$full_command" =~ git\ branch\ ([^[:space:]]+) ]]; then
    branch_name="${BASH_REMATCH[1]}"
fi

if [[ -z "$branch_name" ]]; then
    exit 0
fi

convention_regex="^(feat|fix|refactor|docs|test|chore)/.+"

if [[ ! "$branch_name" =~ $convention_regex ]]; then
  echo "❌ Invalid branch name: '$branch_name'"
  echo "   Branch names must follow the convention: type/description"
  echo "   Example: feat/new-login-page"
  echo "   (To bypass this check, use the --no-verify flag)"
  exit 1
fi

echo "✓ Branch name '$branch_name' is valid."
exit 0
