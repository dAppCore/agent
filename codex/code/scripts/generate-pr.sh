#!/bin/bash
set -euo pipefail

# Default values
DRAFT_FLAG=""
REVIEWERS=""

# Parse arguments
while [[ $# -gt 0 ]]; do
  case "$1" in
    --draft)
      DRAFT_FLAG="--draft"
      shift
      ;;
    --reviewer)
      if [[ -n "$2" ]]; then
        REVIEWERS="$REVIEWERS --reviewer $2"
        shift
        shift
      else
        echo "Error: --reviewer flag requires an argument." >&2
        exit 1
      fi
      ;;
    *)
      echo "Unknown option: $1" >&2
      exit 1
      ;;
  esac
done

# --- Git data ---
# Get default branch (main or master)
DEFAULT_BRANCH=$(git remote show origin | grep 'HEAD branch' | cut -d' ' -f5)
if [[ -z "$DEFAULT_BRANCH" ]]; then
    # Fallback if remote isn't set up or is weird
    if git show-ref --verify --quiet refs/heads/main; then
        DEFAULT_BRANCH="main"
    else
        DEFAULT_BRANCH="master"
    fi
fi

# Get current branch
CURRENT_BRANCH=$(git rev-parse --abbrev-ref HEAD)
if [[ "$CURRENT_BRANCH" == "HEAD" ]]; then
    echo "Error: Not on a branch. Aborting." >&2
    exit 1
fi

# Get merge base
MERGE_BASE=$(git merge-base HEAD "$DEFAULT_BRANCH")
if [[ -z "$MERGE_BASE" ]]; then
    echo "Error: Could not find a common ancestor with '$DEFAULT_BRANCH'. Are you up to date?" >&2
    exit 1
fi


# --- PR Content Generation ---

# Generate Title
# Convert branch name from kebab-case/snake_case to Title Case
TITLE=$(echo "$CURRENT_BRANCH" | sed -E 's/^[a-z-]+\///' | sed -e 's/[-_]/ /g' -e 's/\b\(.\)/\u\1/g')

# Get list of commits
COMMITS=$(git log "$MERGE_BASE"..HEAD --pretty=format:"- %s" --reverse)

# Get list of changed files
CHANGED_FILES=$(git diff --name-only "$MERGE_BASE"..HEAD)

# --- PR Body ---
BODY=$(cat <<EOF
## Summary
$COMMITS

## Changes
\`\`\`
$CHANGED_FILES
\`\`\`

## Test Plan
- [ ] TODO
EOF
)

# --- Create PR ---
echo "Generating PR..." >&2
echo "Title: $TITLE" >&2
echo "---" >&2
echo "$BODY" >&2
echo "---" >&2

# The command to be executed by the plugin runner
gh pr create --title "$TITLE" --body "$BODY" $DRAFT_FLAG $REVIEWERS
