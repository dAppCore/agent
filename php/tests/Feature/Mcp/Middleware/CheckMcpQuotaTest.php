<?php

// SPDX-License-Identifier: EUPL-1.2

declare(strict_types=1);

use Core\Mod\Agentic\Mcp\Services\McpQuotaService;
use Core\Mod\Agentic\Website\Mcp\Middleware\CheckMcpQuota;
use Illuminate\Http\Request;
use Illuminate\Support\Facades\Cache;

beforeEach(function (): void {
    Cache::flush();
    config()->set('mcp.quota_limit', 3);
    config()->set('mcp.quota_period', 'minute');
});

test('CheckMcpQuota_handle_Good_consumes_workspace_quota_and_sets_response_headers', function (): void {
    $workspace = createWorkspace();
    $service = new McpQuotaService;
    $service->setQuota($workspace->id, 3);
    $middleware = new CheckMcpQuota($service);

    $request = Request::create('/api/v1/mcp/tools/call', 'POST');
    $request->attributes->set('workspace_id', $workspace->id);

    $response = $middleware->handle($request, fn () => response()->json(['success' => true]));

    expect($response->getStatusCode())->toBe(200)
        ->and($service->currentUsage($workspace->id))->toBe(1)
        ->and($response->headers->get('X-MCP-Quota-Limit'))->toBe('3')
        ->and($response->headers->get('X-MCP-Quota-Used'))->toBe('1')
        ->and($response->headers->get('X-MCP-Quota-Remaining'))->toBe('2');
});

test('CheckMcpQuota_handle_Bad_rejects_workspaces_that_have_exhausted_their_quota', function (): void {
    $workspace = createWorkspace();
    $service = new McpQuotaService;
    $service->setQuota($workspace->id, 1);
    $service->consume($workspace->id);
    $middleware = new CheckMcpQuota($service);

    $request = Request::create('/api/v1/mcp/tools/call', 'POST');
    $request->attributes->set('workspace_id', $workspace->id);

    $response = $middleware->handle($request, fn () => response()->json(['success' => true]));
    $data = json_decode((string) $response->getContent(), true);

    expect($response->getStatusCode())->toBe(429)
        ->and($data['error'])->toBe('quota_exceeded');
});

test('CheckMcpQuota_handle_Ugly_skips_quota_checks_when_workspace_context_is_absent', function (): void {
    $service = new McpQuotaService;
    $middleware = new CheckMcpQuota($service);
    $request = Request::create('/api/v1/mcp/tools/call', 'POST');

    $response = $middleware->handle($request, fn () => response()->json(['success' => true]));

    expect($response->getStatusCode())->toBe(200)
        ->and($request->attributes->has('mcp_quota'))->toBeFalse();
});
