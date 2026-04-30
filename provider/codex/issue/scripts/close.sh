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
    echo "Usage: /core:issue close <issue-number>"
    exit 1
fi

ISSUE_NUMBER=$1
CURRENT_BRANCH=$(git rev-parse --abbrev-ref HEAD)

echo "Closing #${ISSUE_NUMBER}..."
echo ""

# Get issue title
ISSUE_TITLE=$(gh issue view "${ISSUE_NUMBER}" --json title -q .title)
if [ -z "$ISSUE_TITLE" ]; then
    echo "Could not find issue #${ISSUE_NUMBER}."
    exit 1
fi

echo "Commits on branch '${CURRENT_BRANCH}':"
git log --oneline main..HEAD
echo ""

read -p "Create PR? [Y/n] " -r
echo
if [[ ! $REPLY =~ ^[Nn]$ ]]; then
    gh pr create --title "${ISSUE_TITLE}" --body "Closes #${ISSUE_NUMBER}" --base main
    echo ""
fi

read -p "Comment on issue? [Y/n] " -r
echo
if [[ ! $REPLY =~ ^[Nn]$ ]]; then
    PR_URL=$(gh pr view --json url -q .url)
    if [ -n "$PR_URL" ]; then
        gh issue comment "${ISSUE_NUMBER}" --body "Fixed in ${PR_URL}"
        echo "Commented on issue #${ISSUE_NUMBER}"
    else
        echo "Could not find a pull request for this branch."
    fi
fi
