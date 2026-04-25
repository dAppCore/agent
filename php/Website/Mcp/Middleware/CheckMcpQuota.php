<?php

// SPDX-License-Identifier: EUPL-1.2

declare(strict_types=1);

namespace Core\Mod\Agentic\Website\Mcp\Middleware;

use Closure;
use Core\Mod\Agentic\Mcp\Data\QuotaResult;
use Core\Mod\Agentic\Mcp\Services\McpQuotaService;
use Illuminate\Http\Request;
use Symfony\Component\HttpFoundation\Response;

class CheckMcpQuota
{
    public function __construct(
        protected McpQuotaService $quotaService,
    ) {}

    public function handle(Request $request, Closure $next): Response
    {
        $workspaceId = $this->extractWorkspaceId($request);
        if ($workspaceId === null) {
            return $next($request);
        }

        $period = (string) config('mcp.quota_period', 'minute');
        $quota = $this->quotaService->checkQuota($workspaceId, $period);

        if ($quota->exceeded) {
            return response()->json([
                'error' => 'quota_exceeded',
                'message' => 'MCP workspace quota exceeded.',
                'quota' => $quota->toArray(),
            ], 429, $this->quotaHeaders($quota));
        }

        $updatedQuota = $this->quotaService->consume($workspaceId, 1, $period);
        $request->attributes->set('mcp_quota', $updatedQuota);

        $response = $next($request);
        foreach ($this->quotaHeaders($updatedQuota) as $header => $value) {
            $response->headers->set($header, $value);
        }

        return $response;
    }

    protected function extractWorkspaceId(Request $request): int|string|null
    {
        $workspaceId = $request->attributes->get('workspace_id');
        if (is_int($workspaceId) || is_string($workspaceId)) {
            return $workspaceId;
        }

        $context = $request->attributes->get('mcp_workspace_context');
        if (is_array($context) && isset($context['workspace_id'])) {
            return $context['workspace_id'];
        }

        $workspace = $request->attributes->get('workspace') ?? $request->attributes->get('mcp_workspace');
        if (is_object($workspace) && isset($workspace->id)) {
            return $workspace->id;
        }

        return null;
    }

    /**
     * @return array<string, string>
     */
    protected function quotaHeaders(QuotaResult $quota): array
    {
        return [
            'X-MCP-Quota-Limit' => (string) $quota->limit,
            'X-MCP-Quota-Used' => (string) $quota->used,
            'X-MCP-Quota-Remaining' => (string) $quota->remaining,
            'X-MCP-Quota-Period' => $quota->period,
            'X-MCP-Quota-Reset-At' => $quota->resetAt->toIso8601String(),
        ];
    }
}
