<?php

declare(strict_types=1);

use Illuminate\Database\Migrations\Migration;
use Illuminate\Database\Schema\Blueprint;
use Illuminate\Support\Facades\Schema;

return new class extends Migration
{
    public function up(): void
    {
        Schema::connection('brain')->table('brain_memories', function (Blueprint $table) {
            $table->string('source', 100)->nullable()->after('confidence')
                ->comment('Origin: manual, ingest:claude-md, ingest:plans, etc.');
            $table->index('source');
        });
    }

    public function down(): void
    {
        Schema::connection('brain')->table('brain_memories', function (Blueprint $table) {
            $table->dropColumn('source');
        });
    }
};
