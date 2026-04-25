<?php

// SPDX-License-Identifier: EUPL-1.2

declare(strict_types=1);

use Carbon\CarbonImmutable;
use Core\Mod\Agentic\Mcp\Services\QueryAuditService;
use Illuminate\Database\Schema\Blueprint;
use Illuminate\Support\Facades\Schema;

beforeEach(function (): void {
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

test('QueryAuditService_log_query_aggregate_Good_persists_rows_and_summarises_them_by_period', function (): void {
    $service = new QueryAuditService;

    $service->log('select * from agent_plans', [
        'workspace_id' => 'workspace-1',
        'tool_name' => 'plan_list',
        'result_count' => 2,
        'duration_ms' => 30,
        'recorded_at' => CarbonImmutable::parse('2026-04-25 10:05:00'),
    ]);

    $service->log('select * from agent_plans where slug = ?', [
        'workspace_id' => 'workspace-1',
        'tool_name' => 'plan_get',
        'result_count' => 1,
        'duration_ms' => 10,
        'recorded_at' => CarbonImmutable::parse('2026-04-25 10:15:00'),
    ]);

    $entries = $service->query(['workspace_id' => 'workspace-1']);
    $aggregate = $service->aggregate(['hour']);

    expect($entries)->toHaveCount(2)
        ->and($entries->first()->workspaceId)->toBe('workspace-1')
        ->and($aggregate['hour'])->toHaveCount(1)
        ->and($aggregate['hour'][0]['bucket'])->toBe('2026-04-25 10:00')
        ->and($aggregate['hour'][0]['total'])->toBe(2)
        ->and($aggregate['hour'][0]['safe'])->toBe(2)
        ->and($aggregate['hour'][0]['unsafe'])->toBe(0)
        ->and($aggregate['hour'][0]['average_duration_ms'])->toBe(20)
        ->and($aggregate['hour'][0]['result_count'])->toBe(3);
});

test('QueryAuditService_log_Bad_flags_unsafe_queries_and_filters_them_back_out', function (): void {
    $service = new QueryAuditService;

    $entry = $service->log('DELETE FROM agent_plans WHERE id = 7', [
        'workspace_id' => 'workspace-2',
        'tool_name' => 'plan_delete',
    ]);

    $entries = $service->query(['safe' => false]);

    expect($entry->isSafe)->toBeFalse()
        ->and($entries)->toHaveCount(1)
        ->and($entries->first()->toolName)->toBe('plan_delete');
});

test('QueryAuditService_aggregate_Ugly_rejects_unknown_aggregation_periods', function (): void {
    $service = new QueryAuditService;

    $service->aggregate(['fortnight']);
})->throws(InvalidArgumentException::class, 'Unsupported aggregation period [fortnight].');
