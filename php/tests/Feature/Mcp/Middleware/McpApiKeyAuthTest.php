<?php

// SPDX-License-Identifier: EUPL-1.2

declare(strict_types=1);

use Core\Mod\Agentic\Services\AgentApiKeyService;
use Core\Mod\Agentic\Website\Mcp\Middleware\McpApiKeyAuth;
use Core\Tenant\Models\Workspace;
use Illuminate\Http\Request;

test('McpApiKeyAuth_handle_Good_attaches_workspace_context_for_a_valid_mcp_api_key', function (): void {
    $workspace = createWorkspace();
    $apiKey = createApiKey($workspace, 'MCP Key');
    $middleware = new McpApiKeyAuth(app(AgentApiKeyService::class));

    $request = Request::create('/api/v1/mcp/tools/call', 'POST');
    $request->headers->set('Authorization', 'Bearer '.$apiKey->plainTextKey);

    $capturedContext = null;
    $response = $middleware->handle($request, function (Request $authenticatedRequest) use (&$capturedContext) {
        $capturedContext = $authenticatedRequest->attributes->get('mcp_workspace_context');

        return response()->json([
            'workspace_id' => $authenticatedRequest->attributes->get('workspace_id'),
        ]);
    });

    $data = json_decode((string) $response->getContent(), true);

    expect($response->getStatusCode())->toBe(200)
        ->and($capturedContext)->toBeArray()
        ->and($capturedContext['workspace_id'])->toBe($workspace->id)
        ->and($data['workspace_id'])->toBe($workspace->id)
        ->and($response->headers->get('X-MCP-Workspace-ID'))->toBe((string) $workspace->id);
});

test('McpApiKeyAuth_handle_Bad_rejects_requests_without_an_mcp_api_key', function (): void {
    $middleware = new McpApiKeyAuth(app(AgentApiKeyService::class));
    $request = Request::create('/api/v1/mcp/tools/call', 'POST');

    $response = $middleware->handle($request, fn () => response()->json(['success' => true]));
    $data = json_decode((string) $response->getContent(), true);

    expect($response->getStatusCode())->toBe(401)
        ->and($data['error'])->toBe('unauthorised');
});

test('McpApiKeyAuth_handle_Ugly_blocks_api_keys_for_inactive_workspaces', function (): void {
    $workspace = Workspace::factory()->inactive()->create();
    $apiKey = createApiKey($workspace, 'Inactive Workspace Key');
    $middleware = new McpApiKeyAuth(app(AgentApiKeyService::class));

    $request = Request::create('/api/v1/mcp/tools/call', 'POST');
    $request->headers->set('X-MCP-API-Key', (string) $apiKey->plainTextKey);

    $response = $middleware->handle($request, fn () => response()->json(['success' => true]));
    $data = json_decode((string) $response->getContent(), true);

    expect($response->getStatusCode())->toBe(403)
        ->and($data['error'])->toBe('workspace_inactive');
});
