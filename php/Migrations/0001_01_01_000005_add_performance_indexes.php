<?php

declare(strict_types=1);

use Illuminate\Database\Migrations\Migration;
use Illuminate\Database\Schema\Blueprint;
use Illuminate\Support\Facades\Schema;

return new class extends Migration
{
    /**
     * Add performance indexes for frequently queried columns (DB-002).
     *
     * Analysis per acceptance criteria:
     * - agent_sessions.session_id: the ->unique() constraint in migration 000001
     *   creates a unique index (agent_sessions_session_id_unique) which the query
     *   optimiser uses for string lookups. No additional index required.
     * - agent_plans.slug: ->unique() already creates agent_plans_slug_unique; the
     *   plain agent_plans_slug_index added separately is redundant and is dropped.
     *   A compound (workspace_id, slug) index is added for the common routing
     *   pattern: WHERE workspace_id = ? AND slug = ?
     * - agent_workspace_states.key: already indexed via ->index('key') in
     *   migration 000003. No additional index required.
     */
    public function up(): void
    {
        if (Schema::hasTable('agent_plans')) {
            Schema::table('agent_plans', function (Blueprint $table) {
                // Drop the redundant plain slug index. The unique constraint on slug
                // already provides agent_plans_slug_unique, which covers all lookup queries.
                $table->dropIndex('agent_plans_slug_index');

                // Compound index for the common routing pattern:
                // AgentPlan::where('workspace_id', $id)->where('slug', $slug)->first()
                $table->index(['workspace_id', 'slug'], 'agent_plans_workspace_slug_index');
            });
        }
    }

    public function down(): void
    {
        if (Schema::hasTable('agent_plans')) {
            Schema::table('agent_plans', function (Blueprint $table) {
                $table->dropIndex('agent_plans_workspace_slug_index');

                // Restore the redundant slug index that was present before this migration.
                $table->index('slug');
            });
        }
    }
};
