---
name: compare
description: Compare versions between modules and find incompatibilities
args: "[module] [--prod]"
---

# Compare Module Versions

Compares local module versions against remote, and checks for dependency conflicts.

## Usage

```
/code:compare                 # Compare all modules
/code:compare core-tenant     # Compare specific module
/code:compare --prod          # Compare with production
```

## Action

```bash
#!/bin/bash

# Function to compare semantic versions
# Returns:
#   0 if versions are equal
#   1 if version1 > version2
#   2 if version1 < version2
compare_versions() {
    if [ "$1" == "$2" ]; then
        return 0
    fi
    local winner=$(printf "%s\n%s" "$1" "$2" | sort -V | tail -n 1)
    if [ "$winner" == "$1" ]; then
        return 1
    else
        return 2
    fi
}

# Checks if a version is compatible with a Composer constraint.
is_version_compatible() {
    local version=$1
    local constraint=$2
    local base_version
    local operator=""

    if [[ $constraint == \^* ]]; then
        operator="^"
        base_version=${constraint:1}
    elif [[ $constraint == ~* ]]; then
        operator="~"
        base_version=${constraint:1}
    else
        base_version=$constraint
        compare_versions "$version" "$base_version"
        if [ $? -eq 2 ]; then return 1; else return 0; fi
    fi

    compare_versions "$version" "$base_version"
    if [ $? -eq 2 ]; then
        return 1
    fi

    local major minor patch
    IFS='.' read -r major minor patch <<< "$base_version"
    local upper_bound

    if [ "$operator" == "^" ]; then
        if [ "$major" -gt 0 ]; then
            upper_bound="$((major + 1)).0.0"
        elif [ "$minor" -gt 0 ]; then
            upper_bound="0.$((minor + 1)).0"
        else
            upper_bound="0.0.$((patch + 1))"
        fi
    elif [ "$operator" == "~" ]; then
        upper_bound="$major.$((minor + 1)).0"
    fi

    compare_versions "$version" "$upper_bound"
    if [ $? -eq 2 ]; then
        return 0
    else
        return 1
    fi
}

# Parse arguments
TARGET_MODULE=""
ENV_FLAG=""
for arg in "$@"; do
  case $arg in
    --prod)
      ENV_FLAG="--prod"
      ;;
    *)
      if [[ ! "$arg" == --* ]]; then
        TARGET_MODULE="$arg"
      fi
      ;;
  esac
done

# Get module health data
health_data=$(core dev health $ENV_FLAG)

module_data=$(echo "$health_data" | grep -vE '^(Module|━━|Comparing)' | sed '/^$/d' || true)
if [ -z "$module_data" ]; then
  echo "No module data found."
  exit 0
fi

mapfile -t module_lines <<< "$module_data"
remote_versions=$(echo "$module_data" | awk '{print $1, $3}')

echo "Module Version Comparison"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━"
echo ""
echo "Module          Local     Remote    Status"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"

for line in "${module_lines[@]}"; do
  read -r module local_version remote_version _ <<< "$line"
  if [ -n "$TARGET_MODULE" ] && [ "$module" != "$TARGET_MODULE" ]; then
    continue
  fi

  compare_versions "$local_version" "$remote_version"
  case $? in
    0) status="✓" ;;
    1) status="↑ ahead" ;;
    2) status="↓ behind" ;;
  esac

  printf "%-15s %-9s %-9s %s\n" "$module" "$local_version" "$remote_version" "$status"
done

echo ""
echo "Dependency Check:"

for line in "${module_lines[@]}"; do
  read -r module _ <<< "$line"
  if [ -n "$TARGET_MODULE" ] && [ "$module" != "$TARGET_MODULE" ]; then
    continue
  fi

  if [ -f "$module/composer.json" ]; then
    dependencies=$(jq -r '.require? | select(. != null) | to_entries[] | "\(.key)@\(.value)"' "$module/composer.json")

    for dep in $dependencies; do
      dep_name=$(echo "$dep" | cut -d'@' -f1)
      dep_constraint=$(echo "$dep" | cut -d'@' -f2)

      remote_version=$(echo "$remote_versions" | grep "^$dep_name " | awk '{print $2}')

      if [ -n "$remote_version" ]; then
        if ! is_version_compatible "$remote_version" "$dep_constraint"; then
          echo "⚠ $module requires $dep_name $dep_constraint"
          echo "  But production has $remote_version (incompatible)"
          echo "  Either:"
          echo "  - Deploy a compatible version of $dep_name first"
          echo "  - Or adjust the dependency in $module"
        fi
      fi
    done
  fi
done
```
