<?php

declare(strict_types=1);

use Illuminate\Database\Migrations\Migration;
use Illuminate\Database\Schema\Blueprint;
use Illuminate\Support\Facades\Schema;

return new class extends Migration
{
    public function up(): void
    {
        Schema::table('agent_sessions', function (Blueprint $table) {
            $table->renameColumn('uuid', 'session_id');
            $table->renameColumn('last_activity_at', 'last_active_at');
        });

        // Change column type from uuid to string to allow prefixed IDs (sess_...)
        Schema::table('agent_sessions', function (Blueprint $table) {
            $table->string('session_id')->unique()->change();
        });
    }

    public function down(): void
    {
        Schema::table('agent_sessions', function (Blueprint $table) {
            $table->renameColumn('session_id', 'uuid');
            $table->renameColumn('last_active_at', 'last_activity_at');
        });
    }
};
