<?php

// SPDX-License-Identifier: EUPL-1.2

declare(strict_types=1);

use Core\Mod\Agentic\Mcp\Services\McpQuotaService;
use Illuminate\Support\Facades\Cache;

beforeEach(function (): void {
    Cache::flush();
    config()->set('mcp.quota_limit', 3);
    config()->set('mcp.quota_period', 'minute');
});

test('McpQuotaService_checkQuota_consume_reset_Good_tracks_workspace_usage_per_period', function (): void {
    $service = new McpQuotaService;

    $service->setQuota('workspace-alpha', 5);

    $initial = $service->checkQuota('workspace-alpha');
    $consumed = $service->consume('workspace-alpha', 2);
    $reset = $service->reset('workspace-alpha');

    expect($initial->used)->toBe(0)
        ->and($initial->limit)->toBe(5)
        ->and($consumed->used)->toBe(2)
        ->and($consumed->remaining)->toBe(3)
        ->and($consumed->exceeded)->toBeFalse()
        ->and($reset->used)->toBe(0)
        ->and($reset->remaining)->toBe(5);
});

test('McpQuotaService_checkQuota_Bad_marks_exhausted_workspaces_as_exceeded', function (): void {
    $service = new McpQuotaService;

    $service->setQuota('workspace-bravo', 2);
    $service->consume('workspace-bravo', 2);

    $result = $service->checkQuota('workspace-bravo');

    expect($result->used)->toBe(2)
        ->and($result->remaining)->toBe(0)
        ->and($result->exceeded)->toBeTrue();
});

test('McpQuotaService_consume_Ugly_rejects_non_positive_usage_units', function (): void {
    $service = new McpQuotaService;

    $service->consume('workspace-charlie', 0);
})->throws(InvalidArgumentException::class, 'Quota consumption units must be at least 1.');
