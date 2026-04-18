<?php

declare(strict_types=1);

use Illuminate\Database\Migrations\Migration;
use Illuminate\Database\Schema\Blueprint;
use Illuminate\Support\Facades\Schema;

return new class extends Migration
{
    /**
     * Create prompts and prompt_versions tables.
     *
     * Guarded with hasTable() so this migration is idempotent and
     * can coexist with the consolidated app-level migration.
     */
    public function up(): void
    {
        Schema::disableForeignKeyConstraints();

        if (! Schema::hasTable('prompts')) {
            Schema::create('prompts', function (Blueprint $table) {
                $table->id();
                $table->string('name');
                $table->string('category')->nullable();
                $table->text('description')->nullable();
                $table->text('system_prompt')->nullable();
                $table->text('user_template')->nullable();
                $table->json('variables')->nullable();
                $table->string('model')->nullable();
                $table->json('model_config')->nullable();
                $table->boolean('is_active')->default(true);
                $table->timestamps();

                $table->index('category');
                $table->index('is_active');
            });
        }

        if (! Schema::hasTable('prompt_versions')) {
            Schema::create('prompt_versions', function (Blueprint $table) {
                $table->id();
                $table->foreignId('prompt_id')->constrained()->cascadeOnDelete();
                $table->unsignedInteger('version');
                $table->text('system_prompt')->nullable();
                $table->text('user_template')->nullable();
                $table->json('variables')->nullable();
                $table->foreignId('created_by')->nullable()->constrained('users')->nullOnDelete();
                $table->timestamps();

                $table->index(['prompt_id', 'version']);
            });
        }

        Schema::enableForeignKeyConstraints();
    }

    public function down(): void
    {
        Schema::disableForeignKeyConstraints();
        Schema::dropIfExists('prompt_versions');
        Schema::dropIfExists('prompts');
        Schema::enableForeignKeyConstraints();
    }
};
