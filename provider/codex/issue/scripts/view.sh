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
    echo "Usage: /core:issue view <issue-number>"
    exit 1
fi

ISSUE_NUMBER=$1

# View issue details
echo "Fetching details for issue #${ISSUE_NUMBER} from GitHub..."
gh issue view "${ISSUE_NUMBER}"
