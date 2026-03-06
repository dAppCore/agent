#!/bin/bash
set -e

SUBCOMMAND=$1
shift

case $SUBCOMMAND in
  create)
    php artisan make:migration "$@"
    ;;
  run)
    php artisan migrate "$@"
    ;;
  rollback)
    php artisan migrate:rollback "$@"
    ;;
  fresh)
    php artisan migrate:fresh "$@"
    ;;
  status)
    php artisan migrate:status "$@"
    ;;
  from-model)
    MODEL_NAME=$(basename "$1")
    if [ -z "$MODEL_NAME" ]; then
      echo "Error: Model name not provided."
      exit 1
    fi

    MODEL_PATH=$(find . -path "*/src/Core/Models/${MODEL_NAME}.php" -print -quit)
    if [ -z "$MODEL_PATH" ]; then
      echo "Error: Model ${MODEL_NAME}.php not found."
      exit 1
    fi
    echo "Found model: $MODEL_PATH"

    TABLE_NAME=$(echo "$MODEL_NAME" | sed 's/\(.\)\([A-Z]\)/\1_\2/g' | tr '[:upper:]' '[:lower:]')
    TABLE_NAME="${TABLE_NAME}s"

    MODULE_ROOT=$(echo "$MODEL_PATH" | sed 's|/src/Core/Models/.*||')
    MIGRATIONS_DIR="${MODULE_ROOT}/database/migrations"
    if [ ! -d "$MIGRATIONS_DIR" ]; then
        echo "Error: Migrations directory not found at $MIGRATIONS_DIR"
        exit 1
    fi

    TIMESTAMP=$(date +%Y_%m_%d_%H%M%S)
    MIGRATION_FILE="${MIGRATIONS_DIR}/${TIMESTAMP}_create_${TABLE_NAME}_table.php"

    COLUMNS="            \$table->id();\n"

    if grep -q "use BelongsToWorkspace;" "$MODEL_PATH"; then
      COLUMNS+="            \$table->foreignId('workspace_id')->constrained()->cascadeOnDelete();\n"
    fi

    FILLABLE_LINE=$(grep 'protected \$fillable' "$MODEL_PATH" || echo "")
    if [ -n "$FILLABLE_LINE" ]; then
      FILLABLE_FIELDS=$(echo "$FILLABLE_LINE" | grep -oP "\[\K[^\]]*" | sed "s/['\",]//g")
      for field in $FILLABLE_FIELDS; do
        if [[ "$field" != "workspace_id" ]] && [[ "$field" != *_id ]]; then
            COLUMNS+="            \$table->string('$field');\n"
        fi
      done
    fi

    RELATIONS=$(grep -oP 'public function \K[a-zA-Z0-9_]+(?=\(\): BelongsTo)' "$MODEL_PATH" || echo "")
    for rel in $RELATIONS; do
        COLUMNS+="            \$table->foreignId('${rel}_id')->constrained()->cascadeOnDelete();\n"
    done

    COLUMNS+="            \$table->timestamps();"

    MIGRATION_CONTENT=$(cat <<EOF
<?php

declare(strict_types=1);

use Illuminate\Database\Migrations\Migration;
use Illuminate\Database\Schema\Blueprint;
use Illuminate\Support\Facades\Schema;

return new class extends Migration
{
    public function up(): void
    {
        Schema::create('$TABLE_NAME', function (Blueprint \$table) {
\$COLUMNS
        });
    }

    public function down(): void
    {
        Schema::dropIfExists('$TABLE_NAME');
    }
};
EOF
)

    echo -e "$MIGRATION_CONTENT" > "$MIGRATION_FILE"
    echo "Successfully created migration: $MIGRATION_FILE"
    ;;
  *)
    echo "Usage: /core:migrate <subcommand> [arguments]"
    echo "Subcommands: create, run, rollback, fresh, status, from-model"
    exit 1
    ;;
esac
