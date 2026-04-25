<?php

// SPDX-License-Identifier: EUPL-1.2

declare(strict_types=1);

use Core\Mod\Agentic\Mcp\Services\McpQuotaService;
use Core\Mod\Agentic\Mcp\Services\ToolDependencyService;
use Core\Mod\Agentic\Mcp\Services\ToolRegistry;
use Core\Mod\Agentic\Services\AgentApiKeyService;
use Core\Mod\Agentic\Website\Mcp\Middleware\CheckMcpQuota;
use Core\Mod\Agentic\Website\Mcp\Middleware\McpApiKeyAuth;
use Core\Mod\Agentic\Website\Mcp\Middleware\McpAuthenticate;
use Core\Mod\Agentic\Website\Mcp\Middleware\ValidateToolDependencies;
use Core\Mod\Agentic\Website\Mcp\Middleware\ValidateWorkspaceContext;
use Illuminate\Http\Request;
use Illuminate\Support\Facades\Cache;

beforeEach(function (): void {
    Cache::flush();
    config()->set('mcp.quota_limit', 3);
    config()->set('mcp.quota_period', 'minute');
});

test('McpAuthenticate_handle_Good_chains_auth_quota_workspace_and_dependency_validation', function (): void {
    $workspace = createWorkspace();
    $apiKey = createApiKey($workspace, 'Combined Auth Key');

    $registry = new ToolRegistry;
    $registry->register(mcpMiddlewareToolFixture('plan_list', [
        ['type' => 'context_exists', 'key' => 'workspace_id', 'message' => 'Workspace context required.'],
    ]));

    $quotaService = new McpQuotaService;
    $quotaService->setQuota($workspace->id, 3);

    $authenticate = new McpAuthenticate(
        new McpApiKeyAuth(app(AgentApiKeyService::class)),
        new CheckMcpQuota($quotaService),
        new ValidateWorkspaceContext,
        new ValidateToolDependencies(new ToolDependencyService($registry, $this->app)),
    );

    $request = Request::create('/api/v1/mcp/tools/call', 'POST', [
        'tool' => 'plan_list',
        'arguments' => [
            'workspace_id' => $workspace->id,
        ],
    ]);
    $request->headers->set('X-MCP-API-Key', (string) $apiKey->plainTextKey);

    $capturedWorkspaceId = null;
    $response = $authenticate->handle($request, function (Request $authenticatedRequest) use (&$capturedWorkspaceId) {
        $capturedWorkspaceId = $authenticatedRequest->attributes->get('workspace_id');

        return response()->json(['success' => true]);
    });

    expect($response->getStatusCode())->toBe(200)
        ->and($capturedWorkspaceId)->toBe($workspace->id)
        ->and($quotaService->currentUsage($workspace->id))->toBe(1);
});

test('McpAuthenticate_handle_Bad_stops_the_pipeline_when_workspace_quota_is_exhausted', function (): void {
    $workspace = createWorkspace();
    $apiKey = createApiKey($workspace, 'Quota Exhausted Key');
    $quotaService = new McpQuotaService;
    $quotaService->setQuota($workspace->id, 1);
    $quotaService->consume($workspace->id);

    $authenticate = new McpAuthenticate(
        new McpApiKeyAuth(app(AgentApiKeyService::class)),
        new CheckMcpQuota($quotaService),
        new ValidateWorkspaceContext,
        new ValidateToolDependencies(new ToolDependencyService(new ToolRegistry, $this->app)),
    );

    $request = Request::create('/api/v1/mcp/tools/call', 'POST', [
        'tool' => 'plan_list',
        'arguments' => [
            'workspace_id' => $workspace->id,
        ],
    ]);
    $request->headers->set('Authorization', 'Bearer '.$apiKey->plainTextKey);

    $response = $authenticate->handle($request, fn () => response()->json(['success' => true]));
    $data = json_decode((string) $response->getContent(), true);

    expect($response->getStatusCode())->toBe(429)
        ->and($data['error'])->toBe('quota_exceeded');
});

test('McpAuthenticate_handle_Ugly_bubbles_missing_workspace_context_failures_from_the_validation_stage', function (): void {
    $brokenAuth = new class(app(AgentApiKeyService::class)) extends McpApiKeyAuth
    {
        public function handle(Request $request, Closure $next): \Symfony\Component\HttpFoundation\Response
        {
            return $next($request);
        }
    };

    $authenticate = new McpAuthenticate(
        $brokenAuth,
        new CheckMcpQuota(new McpQuotaService),
        new ValidateWorkspaceContext,
        new ValidateToolDependencies(new ToolDependencyService(new ToolRegistry, $this->app)),
    );

    $request = Request::create('/api/v1/mcp/tools/call', 'POST', [
        'tool' => 'plan_list',
    ]);

    $authenticate->handle($request, fn () => response()->json(['success' => true]));
})->throws(RuntimeException::class, 'MCP workspace context is missing.');
