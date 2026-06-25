<?php

declare(strict_types=1);

use Illuminate\Database\Migrations\Migration;
use Illuminate\Database\Schema\Blueprint;
use Illuminate\Support\Facades\Schema;

/**
 * Re-key sync_records off the soon-to-be-dropped fleet_nodes onto the unified
 * agent identity (workspace_id + agent_id). fleet_node_id stays nullable for
 * back-compat until fleet_nodes is removed.
 */
return new class extends Migration
{
    public function up(): void
    {
        Schema::table('sync_records', function (Blueprint $table): void {
            $table->unsignedBigInteger('workspace_id')->nullable()->after('id');
            $table->string('agent_id')->nullable()->after('workspace_id');
            $table->index(['workspace_id', 'agent_id', 'direction']);
        });
    }

    public function down(): void
    {
        Schema::table('sync_records', function (Blueprint $table): void {
            $table->dropIndex(['workspace_id', 'agent_id', 'direction']);
            $table->dropColumn(['workspace_id', 'agent_id']);
        });
    }
};
