<?php

// SPDX-License-Identifier: EUPL-1.2

declare(strict_types=1);

use Illuminate\Database\Migrations\Migration;
use Illuminate\Database\Schema\Blueprint;
use Illuminate\Support\Facades\Schema;

return new class extends Migration
{
    protected $connection = 'brain';

    public function up(): void
    {
        $schema = Schema::connection($this->getConnection());

        if (! $schema->hasTable('brain_memories') || $schema->hasColumn('brain_memories', 'org')) {
            return;
        }

        $schema->table('brain_memories', function (Blueprint $table): void {
            $table->string('org', 128)->nullable()->after('project')->index();
        });
    }

    public function down(): void
    {
        $schema = Schema::connection($this->getConnection());

        if (! $schema->hasTable('brain_memories') || ! $schema->hasColumn('brain_memories', 'org')) {
            return;
        }

        $schema->table('brain_memories', function (Blueprint $table): void {
            $table->dropIndex(['org']);
            $table->dropColumn('org');
        });
    }
};
