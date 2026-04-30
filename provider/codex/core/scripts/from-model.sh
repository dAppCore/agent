#!/bin/bash
set -e

MODEL_NAME=""
MODEL_PATH_PREFIX="app/Models"
MIGRATION_PATH="database/migrations"

# Parse command-line arguments
while [[ "$#" -gt 0 ]]; do
    case $1 in
        --model-path) MODEL_PATH_PREFIX="$2"; shift ;;
        --path) MIGRATION_PATH="$2"; shift ;;
        *) MODEL_NAME="$1" ;;
    esac
    shift
done

if [ -z "$MODEL_NAME" ]; then
  echo "Usage: /core:migrate from-model <ModelName> [--model-path <path>] [--path <path>]"
  exit 1
fi

MODEL_PATH="${MODEL_PATH_PREFIX}/${MODEL_NAME}.php"
TABLE_NAME=$(echo "$MODEL_NAME" | sed 's/\([A-Z]\)/_\L\1/g' | cut -c 2- | sed 's/$/s/')
MIGRATION_NAME="create_${TABLE_NAME}_table"

if [ ! -f "$MODEL_PATH" ]; then
  echo "Model not found at: $MODEL_PATH"
  exit 1
fi

# Generate the migration file
MIGRATION_FILE=$("${CLAUDE_PLUGIN_ROOT}/scripts/create.sh" "$MIGRATION_NAME" --path "$MIGRATION_PATH")

if [ ! -f "$MIGRATION_FILE" ]; then
  echo "Failed to create migration file."
  exit 1
fi

# Parse the model using the PHP script
SCHEMA_JSON=$(core php "${CLAUDE_PLUGIN_ROOT}/scripts/parse-model.php" "$MODEL_PATH")

if echo "$SCHEMA_JSON" | jq -e '.error' > /dev/null; then
  echo "Error parsing model: $(echo "$SCHEMA_JSON" | jq -r '.error')"
  exit 1
fi

# Generate schema definitions from the JSON output
SCHEMA=$(echo "$SCHEMA_JSON" | jq -r '.columns[] |
  "            $table->" + .type + "(\"" + .name + "\")" +
  (if .type == "foreignId" then "->constrained()->onDelete(\"cascade\")" else "" end) + ";" +
  (if .index then "\n            $table->index(\"" + .name + "\");" else "" end)')

# Insert the generated schema into the migration file
awk -v schema="$SCHEMA" '{ sub("// --- AUTO-GENERATED COLUMNS GO HERE ---", schema); print }' "$MIGRATION_FILE" > "$MIGRATION_FILE.tmp" && mv "$MIGRATION_FILE.tmp" "$MIGRATION_FILE"

echo "Generated migration for $MODEL_NAME in $MIGRATION_FILE"
