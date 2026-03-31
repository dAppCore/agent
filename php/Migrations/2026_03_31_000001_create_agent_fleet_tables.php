<?php

declare(strict_types=1);

use Illuminate\Database\Migrations\Migration;
use Illuminate\Database\Schema\Blueprint;
use Illuminate\Support\Facades\Schema;

return new class extends Migration
{
    public function up(): void
    {
        Schema::disableForeignKeyConstraints();

        if (! Schema::hasTable('fleet_nodes')) {
            Schema::create('fleet_nodes', function (Blueprint $table) {
                $table->id();
                $table->foreignId('workspace_id')->nullable()->constrained()->nullOnDelete();
                $table->string('agent_id')->unique();
                $table->string('platform', 64)->default('unknown');
                $table->json('models')->nullable();
                $table->json('capabilities')->nullable();
                $table->string('status', 32)->default('offline');
                $table->json('compute_budget')->nullable();
                $table->unsignedBigInteger('current_task_id')->nullable();
                $table->timestamp('last_heartbeat_at')->nullable();
                $table->timestamp('registered_at')->nullable();
                $table->timestamps();

                $table->index(['workspace_id', 'status']);
            });
        }

        if (! Schema::hasTable('fleet_tasks')) {
            Schema::create('fleet_tasks', function (Blueprint $table) {
                $table->id();
                $table->foreignId('workspace_id')->nullable()->constrained()->nullOnDelete();
                $table->foreignId('fleet_node_id')->nullable()->constrained('fleet_nodes')->nullOnDelete();
                $table->string('repo');
                $table->string('branch')->nullable();
                $table->text('task');
                $table->string('template')->nullable();
                $table->string('agent_model')->nullable();
                $table->string('status', 32)->default('queued');
                $table->json('result')->nullable();
                $table->json('findings')->nullable();
                $table->json('changes')->nullable();
                $table->json('report')->nullable();
                $table->timestamp('started_at')->nullable();
                $table->timestamp('completed_at')->nullable();
                $table->timestamps();

                $table->index(['workspace_id', 'status']);
                $table->index(['fleet_node_id', 'status']);
            });
        }

        if (! Schema::hasTable('credit_entries')) {
            Schema::create('credit_entries', function (Blueprint $table) {
                $table->id();
                $table->foreignId('workspace_id')->nullable()->constrained()->nullOnDelete();
                $table->foreignId('fleet_node_id')->nullable()->constrained('fleet_nodes')->nullOnDelete();
                $table->string('task_type');
                $table->integer('amount');
                $table->integer('balance_after');
                $table->text('description')->nullable();
                $table->timestamps();

                $table->index(['workspace_id', 'fleet_node_id']);
            });
        }

        if (! Schema::hasTable('sync_records')) {
            Schema::create('sync_records', function (Blueprint $table) {
                $table->id();
                $table->foreignId('fleet_node_id')->nullable()->constrained('fleet_nodes')->nullOnDelete();
                $table->string('direction', 16);
                $table->unsignedInteger('payload_size')->default(0);
                $table->unsignedInteger('items_count')->default(0);
                $table->timestamp('synced_at')->nullable();
                $table->timestamps();

                $table->index(['fleet_node_id', 'direction']);
            });
        }

        Schema::enableForeignKeyConstraints();
    }

    public function down(): void
    {
        Schema::disableForeignKeyConstraints();

        Schema::dropIfExists('sync_records');
        Schema::dropIfExists('credit_entries');
        Schema::dropIfExists('fleet_tasks');
        Schema::dropIfExists('fleet_nodes');

        Schema::enableForeignKeyConstraints();
    }
};
