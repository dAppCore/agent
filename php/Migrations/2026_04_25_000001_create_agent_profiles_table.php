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
        if (Schema::hasTable('agent_profiles')) {
            return;
        }

        Schema::create('agent_profiles', function (Blueprint $table): void {
            $table->bigIncrements('id');
            $table->string('name')->unique();
            $table->string('gateway_url');
            $table->text('api_key_cipher');
            $table->string('cost_class', 1);
            $table->json('capability_tags');
            $table->integer('quota_headroom_pct')->default(100);
            $table->boolean('enabled')->default(true);
            $table->timestamp('last_dispatched_at')->nullable();
            $table->timestamps();
        });
    }

    public function down(): void
    {
        Schema::dropIfExists('agent_profiles');
    }
};
