#!/bin/bash
set -e

MIGRATION_NAME=""
MIGRATION_PATH="database/migrations"

# Parse command-line arguments
while [[ "$#" -gt 0 ]]; do
    case $1 in
        --path) MIGRATION_PATH="$2"; shift ;;
        *) MIGRATION_NAME="$1" ;;
    esac
    shift
done

if [ -z "$MIGRATION_NAME" ]; then
  echo "Usage: /core:migrate create <migration_name> [--path <path>]" >&2
  exit 1
fi

# Let artisan create the file in the specified path
core php artisan make:migration "$MIGRATION_NAME" --path="$MIGRATION_PATH" > /dev/null

# Find the newest file in the target directory that matches the name.
FILE_PATH=$(find "$MIGRATION_PATH" -name "*_$MIGRATION_NAME.php" -print -quit)

if [ -f "$FILE_PATH" ]; then
  # Add the workspace_id column and a placeholder for model generation
  awk '1; /->id\(\);/ { print "            \$table->foreignId(\"workspace_id\")->constrained();\n            // --- AUTO-GENERATED COLUMNS GO HERE ---" }' "$FILE_PATH" > "$FILE_PATH.tmp" && mv "$FILE_PATH.tmp" "$FILE_PATH"
  # Output just the path for other scripts
  echo "$FILE_PATH"
else
  echo "ERROR: Could not find created migration file for '$MIGRATION_NAME' in '$MIGRATION_PATH'." >&2
  exit 1
fi
