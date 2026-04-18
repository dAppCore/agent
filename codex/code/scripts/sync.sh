#!/bin/bash

dry_run=false
target_module=""

while [[ $# -gt 0 ]]; do
  case "$1" in
    --dry-run)
      dry_run=true
      shift
      ;;
    *)
      target_module="$1"
      shift
      ;;
  esac
done

if [ ! -f "repos.yaml" ]; then
    echo "Error: repos.yaml not found"
    exit 1
fi

if [ -z "$target_module" ]; then
    # Detect from current directory
    target_module=$(basename "$(pwd)")
fi

echo "Syncing dependents of $target_module..."

# Get version from composer.json
version=$(jq -r '.version // "1.0.0"' "${target_module}/composer.json" 2>/dev/null || echo "1.0.0")

# Find dependents from repos.yaml
dependents=$(yq -r ".repos | to_entries[] | select(.value.depends[]? == \"$target_module\") | .key" repos.yaml 2>/dev/null)

if [ -z "$dependents" ]; then
    echo "No dependents found for $target_module"
    exit 0
fi

echo "Dependents:"
for dep in $dependents; do
    echo "├── $dep"
    if [ "$dry_run" = true ]; then
        echo "│   └── [dry-run] Would update host-uk/$target_module to v$version"
    else
        composer_file="${dep}/composer.json"
        if [ -f "$composer_file" ]; then
            jq --arg pkg "host-uk/$target_module" --arg ver "$version" \
                '.require[$pkg] = $ver' "$composer_file" > "$composer_file.tmp" && \
                mv "$composer_file.tmp" "$composer_file"
            echo "│   └── Updated composer.json"
        fi
    fi
done
