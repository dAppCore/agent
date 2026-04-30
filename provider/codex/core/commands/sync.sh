#!/bin/bash

# Default values
DRY_RUN=false
TARGET_MODULE=""

# --- Argument Parsing ---
for arg in "$@"; do
  case $arg in
    --dry-run)
      DRY_RUN=true
      shift
      ;;
    *)
      if [ -z "$TARGET_MODULE" ]; then
        TARGET_MODULE=$arg
      fi
      shift
      ;;
  esac
done

# --- Module and Path Detection ---

# This script assumes it is being run from the root of a module in a monorepo.
# The dependent modules are expected to be in the parent directory.
PROJECT_ROOT="."
# For testing purposes, we might be in a different structure.
# If in a mock env, this will be overwritten by a composer.json check.
if [ ! -f "$PROJECT_ROOT/composer.json" ]; then
    # We are likely not in the project root. This is for the mock env.
    PROJECT_ROOT=$(pwd) # Set project root to current dir.
fi

# Determine the current module's name
if [ -z "$TARGET_MODULE" ]; then
  if [ -f "$PROJECT_ROOT/composer.json" ]; then
      TARGET_MODULE=$(jq -r '.name' "$PROJECT_ROOT/composer.json" | cut -d'/' -f2)
  else
      TARGET_MODULE=$(basename "$PROJECT_ROOT")
  fi
fi

# Determine the full package name from the source composer.json
if [ -f "$PROJECT_ROOT/composer.json" ]; then
    PACKAGE_NAME=$(jq -r '.name' "$PROJECT_ROOT/composer.json")
else
    # Fallback for when composer.json is not present (e.g. mock env root)
    PACKAGE_NAME="host-uk/$TARGET_MODULE"
fi


echo "Syncing changes from $PACKAGE_NAME..."

# The repos.yaml is expected at the monorepo root, which is one level above the module directory.
REPOS_YAML_PATH="$PROJECT_ROOT/../repos.yaml"
if [ ! -f "$REPOS_YAML_PATH" ]; then
  # Fallback for test env where repos.yaml is in the current dir.
  if [ -f "repos.yaml" ]; then
    REPOS_YAML_PATH="repos.yaml"
  else
    echo "Error: repos.yaml not found at $REPOS_YAML_PATH"
    exit 1
  fi
fi

# --- Dependency Resolution ---

dependents=$(yq -r ".[\"$TARGET_MODULE\"].dependents[]" "$REPOS_YAML_PATH")

if [ -z "$dependents" ]; then
  echo "No dependents found for $TARGET_MODULE in repos.yaml."
  exit 0
fi

echo -e "\nDependents:"

READY_COUNT=0
NEEDS_FIXES_COUNT=0
FAILED_MODULES=""

# --- Synchronization Logic ---

for dep in $dependents; do
  echo "├── $dep"

  MODULE_PATH="$PROJECT_ROOT/../$dep"
  COMPOSER_JSON="$MODULE_PATH/composer.json"
  NEW_VERSION="dev-main"

  if [ ! -d "$MODULE_PATH" ]; then
    echo "│   └── Module directory not found at $MODULE_PATH. Skipping."
    NEEDS_FIXES_COUNT=$((NEEDS_FIXES_COUNT + 1))
    FAILED_MODULES="$FAILED_MODULES $dep"
    continue
  fi

  if [ "$DRY_RUN" = true ]; then
    echo "│   ├── Would update composer.json for $PACKAGE_NAME to version $NEW_VERSION."
    echo "│   ├── Would run 'composer update $PACKAGE_NAME'."
    echo "│   ├── Tests: (skipped in dry-run)"
    echo "│   └── Ready to commit"
    READY_COUNT=$((READY_COUNT + 1))
  else
    jq ".require[\"$PACKAGE_NAME\"] = \"$NEW_VERSION\"" "$COMPOSER_JSON" > "$COMPOSER_JSON.tmp" && mv "$COMPOSER_JSON.tmp" "$COMPOSER_JSON"
    echo "│   ├── Updated composer.json"

    (cd "$MODULE_PATH" && composer update "$PACKAGE_NAME")
    echo "│   ├── Ran 'composer update'"

    if (cd "$MODULE_PATH" && core php test); then
      echo "│   ├── Tests: ✓ passed"
      echo "│   └── Ready to commit"
      READY_COUNT=$((READY_COUNT + 1))
    else
      echo "│   ├── Tests: ✗ failed"
      echo "│   └── Needs attention: See test logs in $MODULE_PATH"
      NEEDS_FIXES_COUNT=$((NEEDS_FIXES_COUNT + 1))
      FAILED_MODULES="$FAILED_MODULES $dep"
    fi
  fi
done

echo -e "\nSummary: $READY_COUNT ready, $NEEDS_FIXES_COUNT needs fixes"

if [ $NEEDS_FIXES_COUNT -gt 0 ]; then
  echo "Modules needing fixes:$FAILED_MODULES"
fi
