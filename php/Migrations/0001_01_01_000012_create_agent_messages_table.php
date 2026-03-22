<?php

declare(strict_types=1);

use Illuminate\Database\Migrations\Migration;
use Illuminate\Database\Schema\Blueprint;
use Illuminate\Support\Facades\Schema;

return new class extends Migration
{
    public function up(): void
    {
        if (! Schema::hasTable('agent_messages')) {
            Schema::create('agent_messages', function (Blueprint $table) {
                $table->id();
                $table->foreignId('workspace_id')->nullable()->constrained()->nullOnDelete();
                $table->string('from_agent', 100);
                $table->string('to_agent', 100);
                $table->text('content');
                $table->string('subject')->nullable();
                $table->timestamp('read_at')->nullable();
                $table->timestamps();

                $table->index(['to_agent', 'read_at']);
                $table->index(['from_agent', 'to_agent', 'created_at']);
            });
        }
    }

    public function down(): void
    {
        Schema::dropIfExists('agent_messages');
    }
};
