<?php

declare(strict_types=1);

use Illuminate\Database\Migrations\Migration;
use Illuminate\Database\Schema\Blueprint;
use Illuminate\Support\Facades\Schema;

return new class extends Migration
{
    /**
     * Create agent plans, phases, and workspace states tables.
     *
     * Guarded with hasTable() so this migration is idempotent and
     * can coexist with the consolidated app-level migration.
     */
    public function up(): void
    {
        Schema::disableForeignKeyConstraints();

        if (! Schema::hasTable('agent_plans')) {
            Schema::create('agent_plans', function (Blueprint $table) {
                $table->id();
                $table->foreignId('workspace_id')->nullable()->constrained()->nullOnDelete();
                $table->string('slug')->unique();
                $table->string('title');
                $table->text('description')->nullable();
                $table->longText('context')->nullable();
                $table->json('phases')->nullable();
                $table->string('status', 32)->default('draft');
                $table->string('current_phase')->nullable();
                $table->json('metadata')->nullable();
                $table->string('source_file')->nullable();
                $table->timestamps();

                $table->index(['workspace_id', 'status']);
                $table->index('slug');
            });
        }

        if (! Schema::hasTable('agent_phases')) {
            Schema::create('agent_phases', function (Blueprint $table) {
                $table->id();
                $table->foreignId('agent_plan_id')->constrained('agent_plans')->cascadeOnDelete();
                $table->unsignedInteger('order')->default(0);
                $table->string('name');
                $table->text('description')->nullable();
                $table->json('tasks')->nullable();
                $table->json('dependencies')->nullable();
                $table->string('status', 32)->default('pending');
                $table->json('completion_criteria')->nullable();
                $table->timestamp('started_at')->nullable();
                $table->timestamp('completed_at')->nullable();
                $table->json('metadata')->nullable();
                $table->timestamps();

                $table->index(['agent_plan_id', 'order']);
                $table->index(['agent_plan_id', 'status']);
            });
        }

        if (! Schema::hasTable('agent_workspace_states')) {
            Schema::create('agent_workspace_states', function (Blueprint $table) {
                $table->id();
                $table->foreignId('agent_plan_id')->constrained('agent_plans')->cascadeOnDelete();
                $table->string('key');
                $table->json('value')->nullable();
                $table->string('type', 32)->default('json');
                $table->text('description')->nullable();
                $table->timestamps();

                $table->unique(['agent_plan_id', 'key']);
                $table->index('key');
            });
        }

        // Add agent_plan_id to agent_sessions if table exists
        if (Schema::hasTable('agent_sessions') && ! Schema::hasColumn('agent_sessions', 'agent_plan_id')) {
            Schema::table('agent_sessions', function (Blueprint $table) {
                $table->foreignId('agent_plan_id')
                    ->nullable()
                    ->constrained('agent_plans')
                    ->nullOnDelete();
            });
        }

        Schema::enableForeignKeyConstraints();
    }

    public function down(): void
    {
        Schema::disableForeignKeyConstraints();

        if (Schema::hasTable('agent_sessions') && Schema::hasColumn('agent_sessions', 'agent_plan_id')) {
            Schema::table('agent_sessions', function (Blueprint $table) {
                $table->dropForeign(['agent_plan_id']);
                $table->dropColumn('agent_plan_id');
            });
        }

        Schema::dropIfExists('agent_workspace_states');
        Schema::dropIfExists('agent_phases');
        Schema::dropIfExists('agent_plans');

        Schema::enableForeignKeyConstraints();
    }
};
