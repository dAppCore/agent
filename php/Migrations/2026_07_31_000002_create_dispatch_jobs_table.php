<?php

// SPDX-License-Identifier: EUPL-1.2

declare(strict_types=1);

use Illuminate\Database\Migrations\Migration;
use Illuminate\Database\Schema\Blueprint;
use Illuminate\Support\Facades\Schema;

return new class extends Migration
{
    /**
     * The table the DispatchJob model has always needed.
     *
     * The model shipped without it. Nothing in this package created
     * dispatch_jobs, so any consumer that touched DispatchJob got a missing
     * table error — the dispatch queue was a model and a service with no
     * storage under them. lthn.ai carried its own copy of this migration, which
     * is how it worked there and only there.
     *
     * The columns match the model's fillable and casts exactly, including the
     * five (created_by, started_at, findings, changes, report) that lthn.ai
     * added later in a follow-up migration — there is no reason to reproduce
     * that split here.
     *
     * A uuid primary key because job ids are minted by whoever enqueues, often
     * on another machine, and must not depend on this table's sequence.
     */
    public function up(): void
    {
        if (Schema::hasTable('dispatch_jobs')) {
            return;
        }

        Schema::create('dispatch_jobs', function (Blueprint $table): void {
            $table->uuid('id')->primary();
            $table->foreignId('workspace_id')->constrained('workspaces')->cascadeOnDelete();
            $table->string('created_by')->nullable();

            $table->string('repo');
            $table->string('org', 100)->nullable();
            $table->text('task');
            $table->string('agent_type', 64)->nullable();
            $table->string('template', 64)->nullable();
            $table->string('branch')->nullable();

            // Low number sorts first, matching the model's ordering.
            $table->unsignedTinyInteger('priority')->default(5);
            $table->json('labels')->nullable();

            $table->string('status', 32)->default('pending');
            $table->string('assigned_agent')->nullable();
            $table->timestamp('assigned_at')->nullable();
            $table->timestamp('started_at')->nullable();
            $table->timestamp('completed_at')->nullable();

            $table->json('result')->nullable();
            $table->json('findings')->nullable();
            $table->json('changes')->nullable();
            $table->json('report')->nullable();
            $table->json('metadata')->nullable();

            $table->timestamps();

            // The claim query: pending work for a workspace, best first.
            $table->index(['workspace_id', 'status', 'priority'], 'dispatch_jobs_workspace_status_priority');
            // "What is this agent working on", asked on every check-in.
            $table->index(['workspace_id', 'assigned_agent'], 'dispatch_jobs_workspace_agent');
        });
    }

    public function down(): void
    {
        Schema::dropIfExists('dispatch_jobs');
    }
};
