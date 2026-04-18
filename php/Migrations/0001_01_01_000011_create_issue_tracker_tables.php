<?php

declare(strict_types=1);

use Illuminate\Database\Migrations\Migration;
use Illuminate\Database\Schema\Blueprint;
use Illuminate\Support\Facades\Schema;

return new class extends Migration
{
    /**
     * Create issues, sprints, and issue_comments tables.
     *
     * Guarded with hasTable() so this migration is idempotent and
     * can coexist with the consolidated app-level migration.
     */
    public function up(): void
    {
        Schema::disableForeignKeyConstraints();

        if (! Schema::hasTable('sprints')) {
            Schema::create('sprints', function (Blueprint $table) {
                $table->id();
                $table->foreignId('workspace_id')->nullable()->constrained()->nullOnDelete();
                $table->string('slug')->unique();
                $table->string('title');
                $table->text('description')->nullable();
                $table->text('goal')->nullable();
                $table->string('status', 32)->default('planning');
                $table->json('metadata')->nullable();
                $table->timestamp('started_at')->nullable();
                $table->timestamp('ended_at')->nullable();
                $table->timestamp('archived_at')->nullable();
                $table->softDeletes();
                $table->timestamps();

                $table->index(['workspace_id', 'status']);
            });
        }

        if (! Schema::hasTable('issues')) {
            Schema::create('issues', function (Blueprint $table) {
                $table->id();
                $table->foreignId('workspace_id')->nullable()->constrained()->nullOnDelete();
                $table->foreignId('sprint_id')->nullable()->constrained('sprints')->nullOnDelete();
                $table->string('slug')->unique();
                $table->string('title');
                $table->text('description')->nullable();
                $table->string('type', 32)->default('task');
                $table->string('status', 32)->default('open');
                $table->string('priority', 32)->default('normal');
                $table->json('labels')->nullable();
                $table->string('assignee')->nullable();
                $table->string('reporter')->nullable();
                $table->json('metadata')->nullable();
                $table->timestamp('closed_at')->nullable();
                $table->timestamp('archived_at')->nullable();
                $table->softDeletes();
                $table->timestamps();

                $table->index(['workspace_id', 'status']);
                $table->index(['workspace_id', 'sprint_id']);
                $table->index(['workspace_id', 'priority']);
                $table->index(['workspace_id', 'type']);
            });
        }

        if (! Schema::hasTable('issue_comments')) {
            Schema::create('issue_comments', function (Blueprint $table) {
                $table->id();
                $table->foreignId('issue_id')->constrained('issues')->cascadeOnDelete();
                $table->string('author');
                $table->text('body');
                $table->json('metadata')->nullable();
                $table->timestamps();

                $table->index('issue_id');
            });
        }

        Schema::enableForeignKeyConstraints();
    }

    public function down(): void
    {
        Schema::disableForeignKeyConstraints();

        Schema::dropIfExists('issue_comments');
        Schema::dropIfExists('issues');
        Schema::dropIfExists('sprints');

        Schema::enableForeignKeyConstraints();
    }
};
