<?php

// SPDX-License-Identifier: EUPL-1.2

declare(strict_types=1);

namespace Core\Mod\Agentic\Website\Mcp\Middleware;

use Closure;
use Illuminate\Http\Request;
use RuntimeException;
use Symfony\Component\HttpFoundation\Response;

class ValidateWorkspaceContext
{
    public function handle(Request $request, Closure $next): Response
    {
        $context = $this->resolveContext($request);
        if ($context === null) {
            throw new RuntimeException('MCP workspace context is missing.');
        }

        $claimedWorkspaceId = $this->extractClaimedWorkspaceId($request);
        if (
            $claimedWorkspaceId !== null
            && (string) $claimedWorkspaceId !== (string) $context['workspace_id']
        ) {
            return response()->json([
                'error' => 'workspace_mismatch',
                'message' => 'Requested workspace does not match the authenticated MCP workspace.',
                'workspace_id' => (string) $context['workspace_id'],
                'requested_workspace_id' => (string) $claimedWorkspaceId,
            ], 403);
        }

        $request->attributes->set('workspace_id', $context['workspace_id']);
        $request->attributes->set('mcp_workspace_context', $context);

        if (array_key_exists('workspace', $context)) {
            $request->attributes->set('workspace', $context['workspace']);
            $request->attributes->set('mcp_workspace', $context['workspace']);
        }

        if (array_key_exists('api_key', $context)) {
            $request->attributes->set('api_key', $context['api_key']);
        }

        return $next($request);
    }

    /**
     * @return array{workspace_id: int|string, workspace?: mixed, api_key?: mixed}|null
     */
    protected function resolveContext(Request $request): ?array
    {
        $existing = $request->attributes->get('mcp_workspace_context');
        if (is_array($existing) && isset($existing['workspace_id'])) {
            return $existing;
        }

        $workspace = $request->attributes->get('workspace') ?? $request->attributes->get('mcp_workspace');
        $apiKey = $request->attributes->get('api_key');
        $workspaceId = $request->attributes->get('workspace_id')
            ?? (is_object($apiKey) && isset($apiKey->workspace_id) ? $apiKey->workspace_id : null)
            ?? (is_object($workspace) && isset($workspace->id) ? $workspace->id : null);

        if (! is_int($workspaceId) && ! is_string($workspaceId)) {
            return null;
        }

        $context = [
            'workspace_id' => $workspaceId,
        ];

        if ($workspace !== null) {
            $context['workspace'] = $workspace;
        }

        if ($apiKey !== null) {
            $context['api_key'] = $apiKey;
        }

        return $context;
    }

    protected function extractClaimedWorkspaceId(Request $request): int|string|null
    {
        $candidates = [
            $request->input('params.arguments.workspace_id'),
            $request->input('arguments.workspace_id'),
            $request->input('params.workspace_id'),
            $request->input('context.workspace_id'),
            $request->input('params.context.workspace_id'),
            $request->input('workspace_id'),
        ];

        foreach ($candidates as $candidate) {
            if (is_int($candidate) || is_string($candidate)) {
                return $candidate;
            }
        }

        return null;
    }
}
