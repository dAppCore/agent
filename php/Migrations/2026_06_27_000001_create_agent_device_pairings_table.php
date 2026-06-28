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
        if (Schema::hasTable('agent_device_pairings')) {
            return;
        }

        Schema::create('agent_device_pairings', function (Blueprint $table): void {
            $table->id();
            $table->string('code', 6)->index();
            $table->unsignedBigInteger('workspace_id')->index();
            $table->unsignedBigInteger('user_id')->nullable()->index();
            $table->string('label')->nullable();
            $table->json('permissions')->nullable();
            $table->unsignedInteger('rate_limit')->default(100);
            $table->timestamp('key_expires_at')->nullable();
            $table->timestamp('expires_at');
            $table->timestamp('consumed_at')->nullable();
            $table->unsignedBigInteger('agent_api_key_id')->nullable();
            $table->timestamps();

            // Look-ups during the exchange are always on a live (unconsumed) code.
            $table->index(['code', 'consumed_at']);
        });
    }

    public function down(): void
    {
        Schema::dropIfExists('agent_device_pairings');
    }
};
