#!/bin/bash
# Smart commit script for /core:commit command

CUSTOM_MESSAGE=""
AMEND_FLAG=""

# Parse arguments
while (( "$#" )); do
  case "$1" in
    --amend)
      AMEND_FLAG="--amend"
      shift
      ;;
    -*)
      echo "Unsupported flag $1" >&2
      exit 1
      ;;
    *)
      # The rest of the arguments are treated as the commit message
      CUSTOM_MESSAGE="$@"
      break
      ;;
  esac
done

# Get staged changes
STAGED_FILES=$(git diff --staged --name-status)

if [[ -z "$STAGED_FILES" ]]; then
  echo "No staged changes to commit."
  exit 0
fi

# Determine commit type and scope
COMMIT_TYPE="chore" # Default to chore
SCOPE=""

# Get just the file paths
STAGED_FILE_PATHS=$(git diff --staged --name-only)

# Determine type from file paths/status
# Order is important here: test and docs are more specific than feat.
if echo "$STAGED_FILE_PATHS" | grep -q -E "(_test\.go|\.test\.js|/tests/|/spec/)"; then
  COMMIT_TYPE="test"
elif echo "$STAGED_FILE_PATHS" | grep -q -E "(\.md|/docs/|README)"; then
  COMMIT_TYPE="docs"
elif echo "$STAGED_FILES" | grep -q "^A"; then
  COMMIT_TYPE="feat"
elif git diff --staged | grep -q -E "^\+.*(fix|bug|issue)"; then
  COMMIT_TYPE="fix"
elif git diff --staged | grep -q -E "^\+.*(refactor|restructure)"; then
  COMMIT_TYPE="refactor"
fi

# Determine scope from the most common path component
if [[ -n "$STAGED_FILE_PATHS" ]]; then
  # Extract the second component of each path (e.g., 'code' from 'claude/code/file.md')
  # This is a decent heuristic for module name.
  # We filter for lines that have a second component.
  POSSIBLE_SCOPES=$(echo "$STAGED_FILE_PATHS" | grep '/' | cut -d/ -f2)

  if [[ -n "$POSSIBLE_SCOPES" ]]; then
    SCOPE=$(echo "$POSSIBLE_SCOPES" | sort | uniq -c | sort -nr | head -n 1 | awk '{print $2}')
  fi
  # If no scope is found (e.g., all files are in root), SCOPE remains empty, which is valid.
fi

# Construct the commit message
if [[ -n "$CUSTOM_MESSAGE" ]]; then
  COMMIT_MESSAGE="$CUSTOM_MESSAGE"
else
  # Auto-generate a descriptive summary
  DIFF_CONTENT=$(git diff --staged)
  # Try to find a function or class name from the diff
  # This is a simple heuristic that can be greatly expanded.
  SUMMARY=$(echo "$DIFF_CONTENT" | grep -E -o "(function|class|def) \w+" | head -n 1 | sed -e 's/function //g' -e 's/class //g' -e 's/def //g')

  if [[ -z "$SUMMARY" ]]; then
    if [[ $(echo "$STAGED_FILE_PATHS" | wc -l) -eq 1 ]]; then
      FIRST_FILE=$(echo "$STAGED_FILE_PATHS" | head -n 1)
      SUMMARY="update $(basename "$FIRST_FILE")"
    else
      SUMMARY="update multiple files"
    fi
  else
    SUMMARY="update $SUMMARY"
  fi

  SUBJECT="$COMMIT_TYPE($SCOPE): $SUMMARY"
  BODY=$(echo "$DIFF_CONTENT" | grep -E "^\+" | sed -e 's/^+//' | head -n 5 | sed 's/^/  - /')
  COMMIT_MESSAGE="$SUBJECT\n\n$BODY"
fi

# Add Co-Authored-By trailer
CO_AUTHOR="Co-Authored-By: Claude <noreply@anthropic.com>"
if ! echo "$COMMIT_MESSAGE" | grep -q "$CO_AUTHOR"; then
  COMMIT_MESSAGE="$COMMIT_MESSAGE\n\n$CO_AUTHOR"
fi

# Execute the commit
git commit $AMEND_FLAG -m "$(echo -e "$COMMIT_MESSAGE")"

if [[ $? -eq 0 ]]; then
  echo "Commit successful."
else
  echo "Commit failed."
  exit 1
fi
