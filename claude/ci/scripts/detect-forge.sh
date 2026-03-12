#!/bin/bash
# Detect CI provider from git remote
# Outputs: "forgejo" or "github" or "unknown"
# Also exports FORGE_API, FORGE_OWNER, FORGE_REPO

REMOTE_URL=$(git remote get-url origin 2>/dev/null)

if echo "$REMOTE_URL" | grep -q "forge.lthn.ai"; then
    echo "forgejo"
    # Extract owner/repo from SSH or HTTPS URL
    # SSH: ssh://git@forge.lthn.ai:2223/core/go.git
    # HTTPS: https://forge.lthn.ai/core/go.git
    OWNER_REPO=$(echo "$REMOTE_URL" | sed -E 's#.*forge\.lthn\.ai[:/]+([0-9]+/)?##; s#\.git$##')
    export FORGE_API="https://forge.lthn.ai/api/v1"
    export FORGE_OWNER=$(echo "$OWNER_REPO" | cut -d'/' -f1)
    export FORGE_REPO=$(echo "$OWNER_REPO" | cut -d'/' -f2)
elif echo "$REMOTE_URL" | grep -qE "github\.com"; then
    echo "github"
else
    echo "unknown"
fi
