<?php

declare(strict_types=1);

use Illuminate\Database\Migrations\Migration;
use Illuminate\Database\Schema\Blueprint;
use Illuminate\Support\Facades\Schema;

/**
 * Drop the workspace_id foreign key from brain_memories.
 *
 * The brain database may be remote (co-located with Qdrant on the homelab),
 * so cross-database FK constraints to the app's workspaces table are not
 * possible. The column stays as a plain indexed integer.
 */
return new class extends Migration
{
    protected $connection = 'brain';

    public function up(): void
    {
        $schema = Schema::connection($this->getConnection());

        if (! $schema->hasTable('brain_memories')) {
            return;
        }

        $schema->table('brain_memories', function (Blueprint $table) {
            try {
                $table->dropForeign(['workspace_id']);
            } catch (\Throwable) {
                // FK doesn't exist — fresh install, nothing to drop.
            }
        });
    }

    public function down(): void
    {
        // Not re-adding the FK — it was only valid when brain and app shared a database.
    }
};
