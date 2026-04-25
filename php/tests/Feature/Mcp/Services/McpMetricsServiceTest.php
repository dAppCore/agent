<?php

// SPDX-License-Identifier: EUPL-1.2

declare(strict_types=1);

require_once dirname(__DIR__).'/Support/bootstrap.php';

mcpRequire('Mcp/Services/McpMetricsService.php');

use Carbon\CarbonImmutable;
use Core\Mcp\Services\McpMetricsService;
use Illuminate\Database\Schema\Blueprint;
use Illuminate\Support\Facades\DB;
use Illuminate\Support\Facades\Schema;

beforeEach(function (): void {
    CarbonImmutable::setTestNow(CarbonImmutable::parse('2026-04-25 12:00:00'));

    Schema::dropIfExists('mcp_tool_call_stats');
    Schema::dropIfExists('mcp_tool_calls');

    Schema::create('mcp_tool_call_stats', function (Blueprint $table): void {
        $table->id();
        $table->date('date');
        $table->string('server_id');
        $table->string('tool_name');
        $table->unsignedInteger('call_count')->default(0);
        $table->unsignedInteger('success_count')->default(0);
        $table->unsignedInteger('error_count')->default(0);
        $table->unsignedInteger('total_duration_ms')->default(0);
        $table->timestamps();
    });

    Schema::create('mcp_tool_calls', function (Blueprint $table): void {
        $table->id();
        $table->string('server_id');
        $table->string('tool_name');
        $table->string('session_id')->nullable();
        $table->boolean('success')->default(true);
        $table->unsignedInteger('duration_ms')->nullable();
        $table->string('error_message')->nullable();
        $table->string('error_code')->nullable();
        $table->string('plan_slug')->nullable();
        $table->timestamps();
    });
});

afterEach(function (): void {
    CarbonImmutable::setTestNow();
});

test('McpMetricsService_getOverview_Good_returns_current_period_dashboard_metrics', function (): void {
    DB::table('mcp_tool_call_stats')->insert([
        ['date' => '2026-04-24', 'server_id' => 'host-hub', 'tool_name' => 'send_email', 'call_count' => 10, 'success_count' => 9, 'error_count' => 1, 'total_duration_ms' => 1000, 'created_at' => now(), 'updated_at' => now()],
        ['date' => '2026-04-25', 'server_id' => 'marketing', 'tool_name' => 'list_posts', 'call_count' => 5, 'success_count' => 5, 'error_count' => 0, 'total_duration_ms' => 750, 'created_at' => now(), 'updated_at' => now()],
    ]);

    $service = new McpMetricsService;
    $overview = $service->getOverview(2);
    $trend = $service->getDailyTrend(2);

    expect($overview['total_calls'])->toBe(15)
        ->and($overview['success_rate'])->toBe(93.3)
        ->and($trend[0]['date'])->toBe('2026-04-24')
        ->and($trend[1]['total_calls'])->toBe(5);
});

test('McpMetricsService_getErrorBreakdown_Bad_groups_errors_and_plan_activity_from_raw_calls', function (): void {
    DB::table('mcp_tool_calls')->insert([
        ['server_id' => 'host-hub', 'tool_name' => 'send_email', 'success' => false, 'duration_ms' => 100, 'error_message' => 'Bad gateway', 'error_code' => '502', 'plan_slug' => 'plan-a', 'created_at' => now(), 'updated_at' => now()],
        ['server_id' => 'host-hub', 'tool_name' => 'send_email', 'success' => false, 'duration_ms' => 120, 'error_message' => 'Bad gateway', 'error_code' => '502', 'plan_slug' => 'plan-a', 'created_at' => now(), 'updated_at' => now()],
        ['server_id' => 'marketing', 'tool_name' => 'list_posts', 'success' => true, 'duration_ms' => 80, 'plan_slug' => 'plan-b', 'created_at' => now(), 'updated_at' => now()],
    ]);

    $service = new McpMetricsService;
    $errors = $service->getErrorBreakdown(7);
    $plans = $service->getPlanActivity(7);

    expect($errors[0]['tool_name'])->toBe('send_email')
        ->and($errors[0]['error_count'])->toBe(2)
        ->and($plans[0]['plan_slug'])->toBe('plan-a')
        ->and($plans[0]['success_rate'])->toBe(0.0);
});

test('McpMetricsService_getToolPerformance_Ugly_uses_linear_interpolation_for_percentiles', function (): void {
    DB::table('mcp_tool_calls')->insert([
        ['server_id' => 'host-hub', 'tool_name' => 'send_email', 'success' => true, 'duration_ms' => 100, 'created_at' => now(), 'updated_at' => now()],
        ['server_id' => 'host-hub', 'tool_name' => 'send_email', 'success' => true, 'duration_ms' => 200, 'created_at' => now(), 'updated_at' => now()],
        ['server_id' => 'host-hub', 'tool_name' => 'send_email', 'success' => true, 'duration_ms' => 300, 'created_at' => now(), 'updated_at' => now()],
        ['server_id' => 'host-hub', 'tool_name' => 'send_email', 'success' => true, 'duration_ms' => 400, 'created_at' => now(), 'updated_at' => now()],
    ]);

    $service = new McpMetricsService;
    $performance = $service->getToolPerformance(7, 1)->first();

    expect($performance['p50_ms'])->toBe(250.0)
        ->and($performance['p95_ms'])->toBe(385.0)
        ->and($performance['p99_ms'])->toBe(397.0);
});
