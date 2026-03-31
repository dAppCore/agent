<?php

declare(strict_types=1);

use Illuminate\Database\Migrations\Migration;
use Illuminate\Database\Schema\Blueprint;
use Illuminate\Support\Facades\Schema;

return new class extends Migration
{
    public function up(): void
    {
        if (! Schema::hasTable('agent_workspace_states') || Schema::hasColumn('agent_workspace_states', 'category')) {
            return;
        }

        Schema::table('agent_workspace_states', function (Blueprint $table) {
            $table->string('category', 64)->default('general');
            $table->index(['agent_plan_id', 'category'], 'agent_workspace_states_plan_category_idx');
        });
    }

    public function down(): void
    {
        if (! Schema::hasTable('agent_workspace_states') || ! Schema::hasColumn('agent_workspace_states', 'category')) {
            return;
        }

        Schema::table('agent_workspace_states', function (Blueprint $table) {
            $table->dropIndex('agent_workspace_states_plan_category_idx');
            $table->dropColumn('category');
        });
    }
};
