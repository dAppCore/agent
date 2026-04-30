#!/bin/bash

# Check if gh is installed
if ! command -v gh &> /dev/null
then
    echo "GitHub CLI (gh) could not be found. Please install it to use this feature."
    echo "Installation instructions: https://github.com/cli/cli#installation"
    exit 1
fi

# List open issues
echo "Fetching open issues from GitHub..."
gh issue list
