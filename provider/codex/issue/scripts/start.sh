#!/bin/bash

# Check if gh is installed
if ! command -v gh &> /dev/null
then
    echo "GitHub CLI (gh) could not be found. Please install it to use this feature."
    echo "Installation instructions: https://github.com/cli/cli#installation"
    exit 1
fi

# Check for issue number argument
if [ -z "$1" ]; then
    echo "Usage: /core:issue start <issue-number>"
    exit 1
fi

ISSUE_NUMBER=$1

echo "Starting work on #${ISSUE_NUMBER}..."

# Get issue title
ISSUE_TITLE=$(gh issue view "${ISSUE_NUMBER}" --json title -q .title)
if [ -z "$ISSUE_TITLE" ]; then
    echo "Could not find issue #${ISSUE_NUMBER}."
    exit 1
fi

# Sanitize the title for the branch name
BRANCH_NAME=$(echo "$ISSUE_TITLE" | tr '[:upper:]' '[:lower:]' | sed -e 's/[^a-z0-9]/-/g' -e 's/--\+/-/g' -e 's/^-//' -e 's/-$//' | cut -c 1-50)

FULL_BRANCH_NAME="fix/${ISSUE_NUMBER}-${BRANCH_NAME}"

# Create and switch to the new branch
git checkout -b "${FULL_BRANCH_NAME}"

echo ""
echo "1. Created branch: ${FULL_BRANCH_NAME}"
echo "2. Loaded issue context into session"
echo ""
echo "Issue details:"
gh issue view "${ISSUE_NUMBER}"
echo ""
echo "Ready to work. Type /core:issue close ${ISSUE_NUMBER} when done."
