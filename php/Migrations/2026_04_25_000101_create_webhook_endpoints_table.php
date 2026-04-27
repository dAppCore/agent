<?php

// SPDX-License-Identifier: EUPL-1.2

declare(strict_types=1);

use Illuminate\Database\Migrations\Migration;
use Illuminate\Database\Schema\Blueprint;
use Illuminate\Support\Facades\Schema;

return new class extends Migration
{
    public function up(): void
    {
        if (Schema::hasTable('webhook_endpoints')) {
            return;
        }

        Schema::create('webhook_endpoints', function (Blueprint $table): void {
            $table->id();
            $table->foreignId('workspace_id')->nullable()->constrained()->nullOnDelete();
            $table->string('url');
            $table->string('secret', 255)->comment('Current signing secret');
            $table->string('previous_secret', 255)->nullable()->comment('Dual-secret rotation fallback');
            $table->timestamp('previous_secret_expires_at')->nullable();
            $table->json('events')->comment('Subscribed event types or ["*"]');
            $table->boolean('is_active')->default(true);
            $table->string('description')->nullable();
            $table->timestamp('last_triggered_at')->nullable();
            $table->unsignedInteger('failure_count')->default(0);
            $table->timestamp('disabled_at')->nullable();
            $table->timestamps();
            $table->softDeletes();

            $table->index(['workspace_id', 'is_active']);
            $table->index(['is_active', 'disabled_at']);
            $table->index('previous_secret_expires_at');
        });
    }

    public function down(): void
    {
        Schema::dropIfExists('webhook_endpoints');
    }
};
