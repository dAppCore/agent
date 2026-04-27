<?php

// SPDX-License-Identifier: EUPL-1.2

declare(strict_types=1);

use Carbon\CarbonImmutable;
use Core\Mod\Agentic\Mcp\Console\PruneMetricsCommand;
use Illuminate\Console\Command;
use Illuminate\Contracts\Console\Kernel;
use Illuminate\Database\Schema\Blueprint;
use Illuminate\Support\Facades\DB;
use Illuminate\Support\Facades\Schema;

beforeEach(function (): void {
    CarbonImmutable::setTestNow(CarbonImmutable::parse('2026-04-25 12:00:00'));

    $this->app->make(Kernel::class)->registerCommand(
        $this->app->make(PruneMetricsCommand::class),
    );

    Schema::dropIfExists('mcp_tool_metrics');
    Schema::create('mcp_tool_metrics', function (Blueprint $table): void {
        $table->id();
        $table->string('tool_id');
        $table->string('workspace_id');
        $table->date('date');
        $table->unsignedInteger('call_count')->default(0);
        $table->unsignedInteger('success_count')->default(0);
        $table->unsignedInteger('error_count')->default(0);
        $table->unsignedInteger('avg_duration_ms')->default(0);
        $table->unsignedInteger('max_duration_ms')->default(0);
        $table->json('total_calls_by_user')->nullable();
        $table->timestamps();
    });
});

afterEach(function (): void {
    CarbonImmutable::setTestNow();
});

function mcpMetricRow(string $toolId, string $workspaceId, string $date, int $callCount): void
{
    DB::table('mcp_tool_metrics')->insert([
        'tool_id' => $toolId,
        'workspace_id' => $workspaceId,
        'date' => $date,
        'call_count' => $callCount,
        'success_count' => max($callCount - 1, 0),
        'error_count' => min($callCount, 1),
        'avg_duration_ms' => 120,
        'max_duration_ms' => 200,
        'total_calls_by_user' => json_encode(['virgil' => $callCount]),
        'created_at' => now(),
        'updated_at' => now(),
    ]);
}

test('PruneMetricsCommand_handle_Good_prunes_stale_metric_rows_and_reports_the_aggregate', function (): void {
    mcpMetricRow('session_start', 'workspace-1', '2026-03-01', 4);
    mcpMetricRow('session_start', 'workspace-1', '2026-03-10', 6);
    mcpMetricRow('session_start', 'workspace-1', '2026-04-20', 9);

    $this->artisan('mcp:prune-metrics', ['--days' => 30])
        ->expectsOutput('Pruned 2 MCP metric record(s) older than 30 day(s) across 1 bucket(s) covering 10 call(s).')
        ->assertSuccessful();

    expect(DB::table('mcp_tool_metrics')->count())->toBe(1)
        ->and(DB::table('mcp_tool_metrics')->value('date'))->toBe('2026-04-20');
});

test('PruneMetricsCommand_handle_Bad_rejects_non_positive_retention_windows', function (): void {
    mcpMetricRow('session_start', 'workspace-1', '2026-03-01', 4);

    $this->artisan('mcp:prune-metrics', ['--days' => 0])
        ->expectsOutput('--days must be a positive integer.')
        ->assertExitCode(Command::FAILURE);

    expect(DB::table('mcp_tool_metrics')->count())->toBe(1);
});

test('PruneMetricsCommand_handle_Ugly_fails_cleanly_when_the_metrics_table_is_missing', function (): void {
    Schema::dropIfExists('mcp_tool_metrics');

    $this->artisan('mcp:prune-metrics')
        ->expectsOutput('The mcp_tool_metrics table is required for metric pruning.')
        ->assertExitCode(Command::FAILURE);
});
