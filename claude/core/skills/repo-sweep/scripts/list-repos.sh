#!/usr/bin/env bash
# List repos from repos.yaml for sweep targeting.
# Usage: ./list-repos.sh [org-filter]
# Example: ./list-repos.sh core    # only core/* repos
#          ./list-repos.sh         # all repos

REPOS_YAML="${CORE_WORKSPACE:-$HOME/Code/.core}/repos.yaml"

if [ ! -f "$REPOS_YAML" ]; then
    echo "repos.yaml not found at $REPOS_YAML" >&2
    exit 1
fi

ORG_FILTER="${1:-}"

# Extract repo entries (name field from YAML)
if command -v yq &>/dev/null; then
    if [ -n "$ORG_FILTER" ]; then
        yq eval ".repos[] | select(.org == \"$ORG_FILTER\") | .name" "$REPOS_YAML" 2>/dev/null
    else
        yq eval '.repos[].name' "$REPOS_YAML" 2>/dev/null
    fi
else
    # Fallback: grep-based extraction
    grep -E "^\s+name:" "$REPOS_YAML" | sed 's/.*name:\s*//' | sort
fi
