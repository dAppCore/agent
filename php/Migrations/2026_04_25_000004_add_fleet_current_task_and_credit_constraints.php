<?php

// SPDX-License-Identifier: EUPL-1.2

declare(strict_types=1);

use Illuminate\Database\Migrations\Migration;
use Illuminate\Database\Schema\Blueprint;
use Illuminate\Support\Facades\Schema;

return new class extends Migration
{
    public function up(): void
    {
        if (Schema::hasTable('credit_entries') && Schema::hasColumn('credit_entries', 'fleet_task_id')) {
            Schema::table('credit_entries', function (Blueprint $table): void {
                $table->unique(['fleet_node_id', 'fleet_task_id']);
            });
        }

        $driver = Schema::getConnection()->getDriverName();
        if ($driver === 'sqlite') {
            return;
        }

        if (Schema::hasTable('fleet_nodes') && Schema::hasTable('fleet_tasks') && Schema::hasColumn('fleet_nodes', 'current_task_id')) {
            Schema::table('fleet_nodes', function (Blueprint $table): void {
                $table->foreign('current_task_id')
                    ->references('id')
                    ->on('fleet_tasks')
                    ->nullOnDelete();
            });
        }

        if (Schema::hasTable('credit_entries') && Schema::hasTable('fleet_tasks') && Schema::hasColumn('credit_entries', 'fleet_task_id')) {
            Schema::table('credit_entries', function (Blueprint $table): void {
                $table->foreign('fleet_task_id')
                    ->references('id')
                    ->on('fleet_tasks')
                    ->nullOnDelete();
            });
        }
    }

    public function down(): void
    {
        if (Schema::hasTable('credit_entries')) {
            Schema::table('credit_entries', function (Blueprint $table): void {
                $table->dropUnique(['fleet_node_id', 'fleet_task_id']);
            });
        }

        $driver = Schema::getConnection()->getDriverName();
        if ($driver === 'sqlite') {
            return;
        }

        if (Schema::hasTable('fleet_nodes')) {
            Schema::table('fleet_nodes', function (Blueprint $table): void {
                $table->dropForeign(['current_task_id']);
            });
        }

        if (Schema::hasTable('credit_entries') && Schema::hasColumn('credit_entries', 'fleet_task_id')) {
            Schema::table('credit_entries', function (Blueprint $table): void {
                $table->dropForeign(['fleet_task_id']);
            });
        }
    }
};
