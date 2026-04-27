<?php

// SPDX-License-Identifier: EUPL-1.2

declare(strict_types=1);

use Illuminate\Database\Migrations\Migration;
use Illuminate\Database\Schema\Blueprint;
use Illuminate\Support\Facades\DB;
use Illuminate\Support\Facades\Schema;

return new class extends Migration
{
    public function up(): void
    {
        if (! Schema::hasTable('credit_entries')) {
            return;
        }

        if (! Schema::hasColumn('credit_entries', 'agent_id')) {
            Schema::table('credit_entries', function (Blueprint $table): void {
                $table->string('agent_id')->nullable();
                $table->index('agent_id');
            });
        }

        if (! Schema::hasColumn('credit_entries', 'fleet_task_id')) {
            Schema::table('credit_entries', function (Blueprint $table): void {
                $table->unsignedBigInteger('fleet_task_id')->nullable();
                $table->index('fleet_task_id');
            });
        }

        DB::table('credit_entries')
            ->select(['id', 'fleet_node_id'])
            ->whereNull('agent_id')
            ->whereNotNull('fleet_node_id')
            ->orderBy('id')
            ->chunkById(100, function ($entries): void {
                foreach ($entries as $entry) {
                    $agentId = DB::table('fleet_nodes')
                        ->where('id', $entry->fleet_node_id)
                        ->value('agent_id');

                    if (is_string($agentId) && $agentId !== '') {
                        DB::table('credit_entries')
                            ->where('id', $entry->id)
                            ->update(['agent_id' => $agentId]);
                    }
                }
            });
    }

    public function down(): void
    {
        if (! Schema::hasTable('credit_entries')) {
            return;
        }

        if (Schema::hasColumn('credit_entries', 'fleet_task_id')) {
            Schema::table('credit_entries', function (Blueprint $table): void {
                $table->dropIndex(['fleet_task_id']);
                $table->dropColumn('fleet_task_id');
            });
        }

        if (Schema::hasColumn('credit_entries', 'agent_id')) {
            Schema::table('credit_entries', function (Blueprint $table): void {
                $table->dropIndex(['agent_id']);
                $table->dropColumn('agent_id');
            });
        }
    }
};
