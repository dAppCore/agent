<?php

declare(strict_types=1);

use Illuminate\Database\Migrations\Migration;
use Illuminate\Database\Schema\Blueprint;
use Illuminate\Support\Facades\Schema;

/**
 * What the MCP tools did, and how often.
 *
 * McpMetricsService names both tables, McpMonitorCommand exports them as
 * Prometheus counters, and the Agents > Tools pages read them — but the only
 * place either table was ever created was the service's own test, in a
 * beforeEach. So the feature worked under test and answered "Base table or
 * view not found" everywhere else.
 *
 * Two tables on purpose. `mcp_tool_calls` is the log — one row per call, kept
 * for the detail view and for working out why something failed. The stats
 * table is the rolled-up daily counter the dashboards read, so a year of
 * traffic is a page of rows rather than a table scan. The log can be pruned
 * without losing the history.
 *
 * The columns are taken from the test that has been defining this schema
 * implicitly, so the two agree by construction rather than by luck.
 */
return new class extends Migration
{
    public function up(): void
    {
        if (! Schema::hasTable('mcp_tool_call_stats')) {
            Schema::create('mcp_tool_call_stats', function (Blueprint $table): void {
                $table->id();

                $table->date('date');
                $table->string('server_id');
                $table->string('tool_name');

                $table->unsignedInteger('call_count')->default(0);
                $table->unsignedInteger('success_count')->default(0);
                $table->unsignedInteger('error_count')->default(0);
                // Summed, not averaged: an average cannot be added to another
                // average, and every rollup here is an addition.
                $table->unsignedInteger('total_duration_ms')->default(0);

                $table->timestamps();

                // One row per tool per server per day — the grain the counters
                // are incremented at, so it has to be unique or two writers
                // produce two rows and every number halves.
                $table->unique(['date', 'server_id', 'tool_name'], 'mcp_tool_call_stats_grain');
                $table->index(['date', 'tool_name']);
            });
        }

        if (Schema::hasTable('mcp_tool_calls')) {
            return;
        }

        Schema::create('mcp_tool_calls', function (Blueprint $table): void {
            $table->id();

            $table->string('server_id');
            $table->string('tool_name');
            $table->string('session_id')->nullable();

            $table->boolean('success')->default(true);
            $table->unsignedInteger('duration_ms')->nullable();

            // Both, because a code is what you group by and a message is what
            // you read. Neither substitutes for the other.
            $table->string('error_message')->nullable();
            $table->string('error_code')->nullable();

            // Which plan the caller was on when they called. Recorded at call
            // time rather than joined later: a customer changes plan, and the
            // question is what they were entitled to then.
            $table->string('plan_slug')->nullable();

            $table->timestamps();

            $table->index(['tool_name', 'created_at']);
            $table->index(['success', 'created_at']);
            $table->index('session_id');
        });
    }

    public function down(): void
    {
        Schema::dropIfExists('mcp_tool_calls');
        Schema::dropIfExists('mcp_tool_call_stats');
    }
};
