<?php

// SPDX-License-Identifier: EUPL-1.2

declare(strict_types=1);

use Illuminate\Database\Migrations\Migration;
use Illuminate\Database\Schema\Blueprint;
use Illuminate\Support\Facades\Schema;

return new class extends Migration
{
    /** Use the dedicated brain connection so the table lands in the right database. */
    protected $connection = 'brain';

    /**
     * Create brain_memories table for OpenBrain shared knowledge store.
     *
     * Guarded with hasTable() so this migration is idempotent and
     * can coexist with the consolidated app-level migration.
     *
     * No FK to workspaces — the brain database may be remote and
     * the workspaces table only exists in the app database.
     */
    public function up(): void
    {
        $schema = Schema::connection($this->getConnection());

        if (! $schema->hasTable('brain_memories')) {
            $schema->create('brain_memories', function (Blueprint $table) {
                $table->uuid('id')->primary();
                $table->unsignedBigInteger('workspace_id');
                $table->string('agent_id', 64);
                $table->string('type', 32)->index();
                $table->text('content');
                $table->json('tags')->nullable();
                $table->string('project', 128)->nullable()->index();
                $table->float('confidence')->default(0.8);
                $table->uuid('supersedes_id')->nullable();
                $table->timestamp('expires_at')->nullable();
                $table->timestamps();
                $table->softDeletes();

                $table->index('workspace_id');
                $table->index('agent_id');
                $table->index(['workspace_id', 'type']);
                $table->index(['workspace_id', 'project']);
            });

            // Self-referential FK added AFTER create so Postgres sees the
            // primary key on `id` when evaluating the constraint. Adding it
            // inside the create{} block ordered the FK before the PK index
            // on some Postgres versions, breaking with:
            //   SQLSTATE[42830]: there is no unique constraint matching
            //   given keys for referenced table "brain_memories"
            $schema->table('brain_memories', function (Blueprint $table) {
                $table->foreign('supersedes_id')
                    ->references('id')
                    ->on('brain_memories')
                    ->nullOnDelete();
            });
        }
    }

    public function down(): void
    {
        Schema::connection($this->getConnection())->dropIfExists('brain_memories');
    }
};
