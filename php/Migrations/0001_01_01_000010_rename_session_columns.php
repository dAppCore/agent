<?php

declare(strict_types=1);

use Illuminate\Database\Migrations\Migration;
use Illuminate\Database\Schema\Blueprint;
use Illuminate\Support\Facades\Schema;

return new class extends Migration
{
    /**
     * Rename the legacy session columns, where they are still present.
     *
     * The create migration (0001_01_01_000001_create_agentic_tables) was later
     * edited to declare the post-rename names directly, so a fresh database
     * arrives here already correct and has no `uuid` / `last_activity_at` left
     * to rename. Each step is guarded accordingly: this migration only has work
     * to do on databases created before that edit. Without the guards every
     * clean install fails here with "Unknown column 'uuid' in 'agent_sessions'".
     */
    public function up(): void
    {
        if (Schema::hasColumn('agent_sessions', 'uuid')) {
            Schema::table('agent_sessions', function (Blueprint $table) {
                $table->renameColumn('uuid', 'session_id');
            });
        }

        if (Schema::hasColumn('agent_sessions', 'last_activity_at')) {
            Schema::table('agent_sessions', function (Blueprint $table) {
                $table->renameColumn('last_activity_at', 'last_active_at');
            });
        }

        // Widen from uuid to string so prefixed identifiers (sess_...) fit.
        // The unique index is declared by the create migration and survives the
        // type change, so re-declaring it here would fail with "Duplicate key
        // name 'agent_sessions_session_id_unique'".
        Schema::table('agent_sessions', function (Blueprint $table) {
            $table->string('session_id')->change();
        });
    }

    public function down(): void
    {
        if (Schema::hasColumn('agent_sessions', 'session_id')) {
            Schema::table('agent_sessions', function (Blueprint $table) {
                $table->renameColumn('session_id', 'uuid');
            });
        }

        if (Schema::hasColumn('agent_sessions', 'last_active_at')) {
            Schema::table('agent_sessions', function (Blueprint $table) {
                $table->renameColumn('last_active_at', 'last_activity_at');
            });
        }
    }
};
