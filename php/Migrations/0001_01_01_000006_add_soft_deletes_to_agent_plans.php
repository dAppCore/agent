<?php

declare(strict_types=1);

use Illuminate\Database\Migrations\Migration;
use Illuminate\Database\Schema\Blueprint;
use Illuminate\Support\Facades\Schema;

return new class extends Migration
{
    /**
     * Add soft delete support and archived_at timestamp to agent_plans.
     *
     * - archived_at: dedicated timestamp for when a plan was archived, used by
     *   the retention cleanup command to determine when to permanently delete.
     * - deleted_at: standard Laravel soft-delete column.
     */
    public function up(): void
    {
        Schema::table('agent_plans', function (Blueprint $table) {
            $table->timestamp('archived_at')->nullable()->after('source_file');
            $table->softDeletes();
        });
    }

    public function down(): void
    {
        Schema::table('agent_plans', function (Blueprint $table) {
            $table->dropColumn('archived_at');
            $table->dropSoftDeletes();
        });
    }
};
