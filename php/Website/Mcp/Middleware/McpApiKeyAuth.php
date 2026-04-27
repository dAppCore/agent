<?php

// SPDX-License-Identifier: EUPL-1.2

declare(strict_types=1);

namespace Core\Mod\Agentic\Website\Mcp\Middleware;

use Closure;
use Core\Mod\Agentic\Models\AgentApiKey;
use Core\Mod\Agentic\Services\AgentApiKeyService;
use Illuminate\Http\Request;
use Symfony\Component\HttpFoundation\Response;

class McpApiKeyAuth
{
    public function __construct(
        protected AgentApiKeyService $apiKeyService,
    ) {}

    public function handle(Request $request, Closure $next): Response
    {
        $token = $this->extractApiKey($request);
        if ($token === null) {
            return $this->unauthorised(
                'MCP API key required. Use Authorization: Bearer <key> or X-MCP-API-Key.',
            );
        }

        $apiKey = AgentApiKey::findByKey($token);
        if (! $apiKey instanceof AgentApiKey) {
            return $this->unauthorised('Invalid MCP API key.');
        }

        if ($apiKey->isRevoked()) {
            return $this->unauthorised('MCP API key has been revoked.');
        }

        if ($apiKey->isExpired()) {
            return $this->unauthorised('MCP API key has expired.');
        }

        $workspace = $apiKey->workspace;
        if ($workspace === null) {
            return $this->forbidden('MCP API key is not attached to a workspace.');
        }

        if (! $this->workspaceIsActive($workspace)) {
            return $this->forbidden('Workspace is inactive.', 'workspace_inactive');
        }

        $clientIp = $request->ip();
        if (
            $clientIp !== null
            && $apiKey->hasIpRestrictions()
            && ! $this->apiKeyService->isIpAllowed($apiKey, $clientIp)
        ) {
            return $this->forbidden(
                'Request IP is not allowed for this MCP API key.',
                'ip_not_allowed',
            );
        }

        $apiKey->recordUsage();
        if ($clientIp !== null) {
            $apiKey->recordLastUsedIp($clientIp);
        }

        $context = [
            'workspace_id' => $apiKey->workspace_id,
            'workspace' => $workspace,
            'api_key' => $apiKey,
        ];

        $request->attributes->set('agent_api_key', $apiKey);
        $request->attributes->set('api_key', $apiKey);
        $request->attributes->set('workspace', $workspace);
        $request->attributes->set('workspace_id', $apiKey->workspace_id);
        $request->attributes->set('mcp_workspace', $workspace);
        $request->attributes->set('mcp_workspace_context', $context);

        $response = $next($request);
        $response->headers->set('X-MCP-Workspace-ID', (string) $apiKey->workspace_id);

        return $response;
    }

    protected function extractApiKey(Request $request): ?string
    {
        $bearerToken = $request->bearerToken();
        if (is_string($bearerToken) && $bearerToken !== '') {
            return $bearerToken;
        }

        $authorisation = $request->header('Authorization');
        if (is_string($authorisation) && str_starts_with($authorisation, 'Bearer ')) {
            return substr($authorisation, 7);
        }

        $headerToken = $request->header('X-MCP-API-Key') ?? $request->header('X-API-Key');
        if (is_string($headerToken) && $headerToken !== '') {
            return $headerToken;
        }

        return null;
    }

    protected function workspaceIsActive(object $workspace): bool
    {
        if (! method_exists($workspace, 'getAttribute')) {
            return true;
        }

        $isActive = $workspace->getAttribute('is_active');
        if ($isActive === null) {
            return true;
        }

        return (bool) $isActive;
    }

    protected function unauthorised(string $message): Response
    {
        return response()->json([
            'error' => 'unauthorised',
            'message' => $message,
        ], 401);
    }

    protected function forbidden(string $message, string $error = 'forbidden'): Response
    {
        return response()->json([
            'error' => $error,
            'message' => $message,
        ], 403);
    }
}
