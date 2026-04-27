<?php

// SPDX-License-Identifier: EUPL-1.2

declare(strict_types=1);

use Core\Mod\Agentic\Website\Mcp\Middleware\ValidateWorkspaceContext;
use Illuminate\Http\Request;

test('ValidateWorkspaceContext_handle_Good_normalises_authenticated_workspace_context', function (): void {
    $workspace = createWorkspace();
    $apiKey = createApiKey($workspace, 'Workspace Context Key');
    $middleware = new ValidateWorkspaceContext;

    $request = Request::create('/api/v1/mcp/tools/call', 'POST', [
        'arguments' => [
            'workspace_id' => $workspace->id,
        ],
    ]);
    $request->attributes->set('workspace_id', $workspace->id);
    $request->attributes->set('workspace', $workspace);
    $request->attributes->set('api_key', $apiKey);

    $capturedContext = null;
    $response = $middleware->handle($request, function (Request $validatedRequest) use (&$capturedContext) {
        $capturedContext = $validatedRequest->attributes->get('mcp_workspace_context');

        return response()->json(['success' => true]);
    });

    expect($response->getStatusCode())->toBe(200)
        ->and($capturedContext)->toBeArray()
        ->and($capturedContext['workspace_id'])->toBe($workspace->id);
});

test('ValidateWorkspaceContext_handle_Bad_rejects_workspace_ids_that_do_not_match_the_authenticated_workspace', function (): void {
    $workspace = createWorkspace();
    $middleware = new ValidateWorkspaceContext;

    $request = Request::create('/api/v1/mcp/tools/call', 'POST', [
        'arguments' => [
            'workspace_id' => $workspace->id + 1,
        ],
    ]);
    $request->attributes->set('workspace_id', $workspace->id);
    $request->attributes->set('workspace', $workspace);

    $response = $middleware->handle($request, fn () => response()->json(['success' => true]));
    $data = json_decode((string) $response->getContent(), true);

    expect($response->getStatusCode())->toBe(403)
        ->and($data['error'])->toBe('workspace_mismatch');
});

test('ValidateWorkspaceContext_handle_Ugly_throws_when_no_authenticated_workspace_context_exists', function (): void {
    $middleware = new ValidateWorkspaceContext;
    $request = Request::create('/api/v1/mcp/tools/call', 'POST');

    $middleware->handle($request, fn () => response()->json(['success' => true]));
})->throws(RuntimeException::class, 'MCP workspace context is missing.');
