<?php

// SPDX-License-Identifier: EUPL-1.2

declare(strict_types=1);

use Illuminate\Database\Migrations\Migration;
use Illuminate\Database\Schema\Blueprint;
use Illuminate\Support\Facades\Schema;

return new class extends Migration
{
    /**
     * The other table this package's models needed and never had.
     *
     * Same story as dispatch_jobs: AgentRegistration ships here, its table did
     * not, and the only place it existed was lthn.ai's own copy. Anything that
     * registered an agent, took a heartbeat, or asked what an agent was working
     * on failed on a missing table.
     *
     * Columns match the model's fillable, which includes the four (platform,
     * models, compute_budget, current_task_id) that lthn.ai bolted on in a
     * later migration. Creating the shape the model actually declares is worth
     * more than reproducing the order it was discovered in.
     */
    public function up(): void
    {
        if (Schema::hasTable('agent_registrations')) {
            return;
        }

        Schema::create('agent_registrations', function (Blueprint $table): void {
            $table->uuid('id')->primary();
            $table->foreignId('workspace_id')->constrained('workspaces')->cascadeOnDelete();

            $table->string('agent_id');
            $table->string('hostname')->nullable();
            $table->string('platform')->nullable();

            $table->json('capabilities')->nullable();
            $table->json('models')->nullable();
            $table->json('compute_budget')->nullable();
            $table->unsignedTinyInteger('max_concurrent')->default(1);
            $table->json('labels')->nullable();
            $table->string('version')->nullable();

            $table->string('status', 32)->default('online');

            // Deliberately not a foreign key to dispatch_jobs: an agent holding
            // a job that is later hard-deleted should be left holding nothing,
            // not cascade-deleted along with it.
            $table->uuid('current_task_id')->nullable();

            $table->json('metadata')->nullable();
            $table->timestamp('connected_at')->nullable();
            $table->timestamp('last_heartbeat_at')->nullable();
            $table->timestamps();

            // One registration per agent name per workspace — re-registering
            // updates rather than duplicating.
            $table->unique(['workspace_id', 'agent_id'], 'agent_registrations_workspace_agent');
            $table->index(['workspace_id', 'status'], 'agent_registrations_workspace_status');
        });
    }

    public function down(): void
    {
        Schema::dropIfExists('agent_registrations');
    }
};
