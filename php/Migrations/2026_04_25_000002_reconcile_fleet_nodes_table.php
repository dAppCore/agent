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
        if (! Schema::hasTable('fleet_nodes') || Schema::hasColumn('fleet_nodes', 'current_task_id')) {
            return;
        }

        Schema::table('fleet_nodes', function (Blueprint $table): void {
            $table->unsignedBigInteger('current_task_id')->nullable();
        });
    }

    public function down(): void
    {
        if (! Schema::hasTable('fleet_nodes') || ! Schema::hasColumn('fleet_nodes', 'current_task_id')) {
            return;
        }

        Schema::table('fleet_nodes', function (Blueprint $table): void {
            $table->dropColumn('current_task_id');
        });
    }
};
