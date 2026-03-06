#!/bin/bash

MODULE_NAME=$1
TARGET_PATH=$2

if [ -z "$MODULE_NAME" ] || [ -z "$TARGET_PATH" ]; then
  echo "Usage: doc-module.sh <ModuleName> <TargetPath>" >&2
  exit 1
fi

MODULE_PATH="${TARGET_PATH}/${MODULE_NAME}"
COMPOSER_JSON_PATH="${MODULE_PATH}/composer.json"

if [ ! -d "$MODULE_PATH" ]; then
  echo "Error: Module directory not found at '$MODULE_PATH'." >&2
  exit 1
fi

if [ ! -f "$COMPOSER_JSON_PATH" ]; then
  echo "Error: 'composer.json' not found in module directory '$MODULE_PATH'." >&2
  exit 1
fi

# --- PARSING & MARKDOWN GENERATION ---
# Use jq to parse the composer.json file.
NAME=$(jq -r '.name' "$COMPOSER_JSON_PATH")
DESCRIPTION=$(jq -r '.description' "$COMPOSER_JSON_PATH")
TYPE=$(jq -r '.type' "$COMPOSER_JSON_PATH")
LICENSE=$(jq -r '.license' "$COMPOSER_JSON_PATH")

echo "# Module: $NAME"
echo ""
echo "**Description:** $DESCRIPTION"
echo "**Type:** $TYPE"
echo "**License:** $LICENSE"
echo ""

# List dependencies
DEPENDENCIES=$(jq -r '.require | keys[] as $key | "\($key): \(.[$key])"' "$COMPOSER_JSON_PATH")
if [ -n "$DEPENDENCIES" ]; then
  echo "## Dependencies"
  echo ""
  echo "$DEPENDENCIES" | while read -r DEP; do
    echo "- $DEP"
  done
  echo ""
fi

# List dev dependencies
DEV_DEPENDENCIES=$(jq -r '.["require-dev"] | keys[] as $key | "\($key): \(.[$key])"' "$COMPOSER_JSON_PATH")
if [ -n "$DEV_DEPENDENCIES" ]; then
  echo "## Dev Dependencies"
  echo ""
  echo "$DEV_DEPENDENCIES" | while read -r DEP; do
    echo "- $DEP"
  done
  echo ""
fi
