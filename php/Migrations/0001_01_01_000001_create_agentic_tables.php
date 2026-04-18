<?php

declare(strict_types=1);

use Illuminate\Database\Migrations\Migration;
use Illuminate\Database\Schema\Blueprint;
use Illuminate\Support\Facades\Schema;

return new class extends Migration
{
    /**
     * Agentic module tables - AI agents, tasks, sessions.
     *
     * Guarded with hasTable() so this migration is idempotent and
     * can coexist with the consolidated app-level migration.
     */
    public function up(): void
    {
        Schema::disableForeignKeyConstraints();

        if (! Schema::hasTable('agent_api_keys')) {
            Schema::create('agent_api_keys', function (Blueprint $table) {
                $table->id();
                $table->foreignId('workspace_id')->nullable()->constrained()->nullOnDelete();
                $table->string('name');
                $table->string('key');
                $table->json('permissions')->nullable();
                $table->unsignedInteger('rate_limit')->nullable();
                $table->unsignedBigInteger('call_count')->default(0);
                $table->timestamp('last_used_at')->nullable();
                $table->timestamp('expires_at')->nullable();
                $table->timestamp('revoked_at')->nullable();
                $table->timestamps();

                $table->index('workspace_id');
                $table->index('key');
            });
        }

        if (! Schema::hasTable('agent_sessions')) {
            Schema::create('agent_sessions', function (Blueprint $table) {
                $table->id();
                $table->foreignId('workspace_id')->nullable()->constrained()->nullOnDelete();
                $table->foreignId('agent_api_key_id')->nullable()->constrained()->nullOnDelete();
                $table->string('session_id')->unique();
                $table->string('agent_type')->nullable();
                $table->string('status')->default('active');
                $table->json('context_summary')->nullable();
                $table->json('work_log')->nullable();
                $table->json('artifacts')->nullable();
                $table->json('handoff_notes')->nullable();
                $table->text('final_summary')->nullable();
                $table->timestamp('started_at')->nullable();
                $table->timestamp('last_active_at')->nullable();
                $table->timestamp('ended_at')->nullable();
                $table->timestamps();

                $table->index('workspace_id');
                $table->index('status');
                $table->index('agent_type');
            });
        }

        Schema::enableForeignKeyConstraints();
    }

    public function down(): void
    {
        Schema::disableForeignKeyConstraints();
        Schema::dropIfExists('agent_sessions');
        Schema::dropIfExists('agent_api_keys');
        Schema::enableForeignKeyConstraints();
    }
};
