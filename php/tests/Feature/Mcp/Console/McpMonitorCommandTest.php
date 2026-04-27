<?php

// SPDX-License-Identifier: EUPL-1.2

declare(strict_types=1);

use Carbon\CarbonImmutable;
use Core\Mod\Agentic\Mcp\Console\McpMonitorCommand;
use Core\Mod\Agentic\Mcp\Services\QueryAuditService;
use Illuminate\Console\Command;
use Illuminate\Contracts\Console\Kernel;
use Illuminate\Database\Schema\Blueprint;
use Illuminate\Support\Facades\Artisan;
use Illuminate\Support\Facades\DB;
use Illuminate\Support\Facades\Schema;

beforeEach(function (): void {
    CarbonImmutable::setTestNow(CarbonImmutable::parse('2026-04-25 12:00:00'));

    $this->app->make(Kernel::class)->registerCommand(
        $this->app->make(McpMonitorCommand::class),
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

    Schema::dropIfExists('mcp_audit_entries');
    Schema::create('mcp_audit_entries', function (Blueprint $table): void {
        $table->id();
        $table->string('workspace_id')->nullable();
        $table->string('tool_name')->nullable();
        $table->longText('query_text');
        $table->string('query_hash', 64);
        $table->boolean('is_safe')->default(true);
        $table->unsignedInteger('result_count')->nullable();
        $table->unsignedInteger('duration_ms')->nullable();
        $table->json('metadata')->nullable();
        $table->timestamps();
    });
});

afterEach(function (): void {
    CarbonImmutable::setTestNow();
});

function mcpMonitorMetric(string $toolId, string $date, int $callCount, int $successCount, int $errorCount, int $avgDurationMs): void
{
    DB::table('mcp_tool_metrics')->insert([
        'tool_id' => $toolId,
        'workspace_id' => 'workspace-1',
        'date' => $date,
        'call_count' => $callCount,
        'success_count' => $successCount,
        'error_count' => $errorCount,
        'avg_duration_ms' => $avgDurationMs,
        'max_duration_ms' => $avgDurationMs + 25,
        'total_calls_by_user' => json_encode(['virgil' => $callCount]),
        'created_at' => now(),
        'updated_at' => now(),
    ]);
}

test('McpMonitorCommand_handle_Good_outputs_a_machine_readable_summary_report', function (): void {
    mcpMonitorMetric('session_start', '2026-04-24', 10, 9, 1, 120);
    mcpMonitorMetric('report_generate', '2026-04-25', 5, 5, 0, 180);

    $exitCode = Artisan::call('mcp:monitor', [
        'action' => 'report',
        '--days' => 7,
        '--json' => true,
    ]);
    $output = json_decode(Artisan::output(), true, 512, JSON_THROW_ON_ERROR);

    expect($exitCode)->toBe(Command::SUCCESS)
        ->and($output['action'])->toBe('report')
        ->and($output['report']['overview']['total_calls'])->toBe(15)
        ->and($output['report']['top_tools'][0]['tool_id'])->toBe('session_start');
});

test('McpMonitorCommand_handle_Bad_rejects_unknown_actions', function (): void {
    $this->artisan('mcp:monitor', ['action' => 'invalid'])
        ->expectsOutput('Unsupported monitor action [invalid].')
        ->assertExitCode(Command::FAILURE);
});

test('McpMonitorCommand_handle_Ugly_returns_failure_when_status_is_critical', function (): void {
    mcpMonitorMetric('send_email', '2026-04-25', 10, 8, 2, 140);
    mcpMonitorMetric('send_email', '2026-04-24', 10, 8, 2, 160);

    app(QueryAuditService::class)->log('select * from agent_plans', [
        'workspace_id' => 'workspace-1',
        'tool_name' => 'plan_list',
        'duration_ms' => 20,
        'result_count' => 2,
    ]);

    $this->artisan('mcp:monitor', ['action' => 'status'])
        ->expectsOutputToContain('MCP Health Status: CRITICAL')
        ->assertExitCode(Command::FAILURE);
});
