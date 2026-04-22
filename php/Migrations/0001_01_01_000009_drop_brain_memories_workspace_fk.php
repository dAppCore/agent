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

        // Laravel Blueprint defers statement execution until the closure
        // returns, so a try/catch INSIDE the closure doesn't catch the
        // deferred SQL's failure. Instead, check the constraint exists
        // before issuing the drop, across both pgsql + mariadb backends.
        $conn = $schema->getConnection();
        $driver = $conn->getDriverName();
        $constraint = 'brain_memories_workspace_id_foreign';

        $exists = match ($driver) {
            'pgsql' => (bool) $conn->selectOne(
                "SELECT 1 FROM information_schema.table_constraints
                 WHERE table_name = 'brain_memories' AND constraint_name = ?",
                [$constraint],
            ),
            'mariadb', 'mysql' => (bool) $conn->selectOne(
                "SELECT 1 FROM information_schema.table_constraints
                 WHERE table_schema = DATABASE() AND table_name = 'brain_memories'
                       AND constraint_name = ?",
                [$constraint],
            ),
            default => false,   // unknown driver — don't attempt the drop
        };

        if ($exists) {
            $schema->table('brain_memories', function (Blueprint $table) {
                $table->dropForeign(['workspace_id']);
            });
        }
    }

    public function down(): void
    {
        // Not re-adding the FK — it was only valid when brain and app shared a database.
    }
};
