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
        if (Schema::hasTable('agent_dispatches')) {
            return;
        }

        Schema::create('agent_dispatches', function (Blueprint $table): void {
            $table->id();
            $table->integer('ticket_id')->index();
            $table->unsignedBigInteger('profile_id')->nullable();
            $table->string('response_id');
            $table->string('run_id')->nullable();
            $table->string('status')->default('queued');
            $table->text('error')->nullable();
            $table->timestamps();
        });
    }

    public function down(): void
    {
        Schema::dropIfExists('agent_dispatches');
    }
};
