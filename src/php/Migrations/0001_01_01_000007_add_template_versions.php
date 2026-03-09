<?php

declare(strict_types=1);

use Illuminate\Database\Migrations\Migration;
use Illuminate\Database\Schema\Blueprint;
use Illuminate\Support\Facades\Schema;

return new class extends Migration
{
    /**
     * Create plan_template_versions table and add template_version_id to agent_plans.
     *
     * Template versions snapshot YAML template content at plan-creation time so
     * existing plans are never affected when a template file is updated.
     *
     * Deduplication: identical content reuses the same version row (same content_hash).
     *
     * Guarded with hasTable()/hasColumn() so this migration is idempotent and
     * can coexist with a consolidated app-level migration.
     */
    public function up(): void
    {
        Schema::disableForeignKeyConstraints();

        if (! Schema::hasTable('plan_template_versions')) {
            Schema::create('plan_template_versions', function (Blueprint $table) {
                $table->id();
                $table->string('slug');
                $table->unsignedInteger('version');
                $table->string('name');
                $table->json('content');
                $table->char('content_hash', 64);
                $table->timestamps();

                $table->unique(['slug', 'version']);
                $table->index(['slug', 'content_hash']);
            });
        }

        if (Schema::hasTable('agent_plans') && ! Schema::hasColumn('agent_plans', 'template_version_id')) {
            Schema::table('agent_plans', function (Blueprint $table) {
                $table->foreignId('template_version_id')
                    ->nullable()
                    ->constrained('plan_template_versions')
                    ->nullOnDelete()
                    ->after('source_file');
            });
        }

        Schema::enableForeignKeyConstraints();
    }

    public function down(): void
    {
        Schema::disableForeignKeyConstraints();

        if (Schema::hasTable('agent_plans') && Schema::hasColumn('agent_plans', 'template_version_id')) {
            Schema::table('agent_plans', function (Blueprint $table) {
                $table->dropForeign(['template_version_id']);
                $table->dropColumn('template_version_id');
            });
        }

        Schema::dropIfExists('plan_template_versions');

        Schema::enableForeignKeyConstraints();
    }
};
