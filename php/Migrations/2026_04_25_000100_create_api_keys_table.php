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
        if (Schema::hasTable('api_keys')) {
            return;
        }

        Schema::create('api_keys', function (Blueprint $table): void {
            $table->id();
            $table->foreignId('workspace_id')->nullable()->constrained()->nullOnDelete();
            $table->unsignedBigInteger('user_id')->nullable()->index();
            $table->string('name');
            $table->string('key', 255)->comment('Bcrypt hash of the secret portion only');
            $table->string('hash_algorithm', 16)->default('bcrypt');
            $table->string('prefix', 11)->comment('hk_xxxxxxxx');
            $table->json('scopes')->nullable();
            $table->json('allowed_ips')->nullable();
            $table->timestamp('last_used_at')->nullable();
            $table->timestamp('expires_at')->nullable();
            $table->timestamp('grace_period_ends_at')->nullable();
            $table->timestamps();
            $table->softDeletes();

            $table->index(['prefix', 'deleted_at']);
            $table->index(['workspace_id', 'expires_at']);
        });
    }

    public function down(): void
    {
        Schema::dropIfExists('api_keys');
    }
};
