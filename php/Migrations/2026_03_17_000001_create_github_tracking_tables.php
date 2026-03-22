<?php

declare(strict_types=1);

use Illuminate\Database\Migrations\Migration;
use Illuminate\Database\Schema\Blueprint;
use Illuminate\Support\Facades\Schema;

return new class extends Migration
{
    public function up(): void
    {
        // Raw webhook events for audit trail
        Schema::create('github_webhook_events', function (Blueprint $table) {
            $table->id();
            $table->string('event', 50)->index();
            $table->string('action', 50)->default('');
            $table->string('repo', 100)->index();
            $table->json('payload');
            $table->timestamp('created_at')->useCurrent();
        });

        // CodeRabbit review results — the KPI table
        Schema::create('coderabbit_reviews', function (Blueprint $table) {
            $table->id();
            $table->string('repo', 100)->index();
            $table->unsignedInteger('pr_number');
            $table->string('result', 30)->index(); // approved, changes_requested
            $table->text('findings')->nullable();   // Review body with findings
            $table->boolean('findings_dispatched')->default(false);
            $table->boolean('findings_resolved')->default(false);
            $table->timestamp('created_at')->useCurrent();
            $table->timestamp('resolved_at')->nullable();

            $table->index(['repo', 'pr_number']);
        });
    }

    public function down(): void
    {
        Schema::dropIfExists('coderabbit_reviews');
        Schema::dropIfExists('github_webhook_events');
    }
};
