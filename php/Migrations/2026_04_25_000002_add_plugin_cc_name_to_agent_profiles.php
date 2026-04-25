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
        if (! Schema::hasTable('agent_profiles')) {
            return;
        }

        if (Schema::hasColumn('agent_profiles', 'plugin_cc_name')) {
            return;
        }

        Schema::table('agent_profiles', function (Blueprint $table): void {
            $table->string('plugin_cc_name')->nullable()->after('name');
        });
    }

    public function down(): void
    {
        if (! Schema::hasTable('agent_profiles')) {
            return;
        }

        if (! Schema::hasColumn('agent_profiles', 'plugin_cc_name')) {
            return;
        }

        Schema::table('agent_profiles', function (Blueprint $table): void {
            $table->dropColumn('plugin_cc_name');
        });
    }
};
