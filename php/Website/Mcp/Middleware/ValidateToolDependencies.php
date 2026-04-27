<?php

// SPDX-License-Identifier: EUPL-1.2

declare(strict_types=1);

namespace Core\Mod\Agentic\Website\Mcp\Middleware;

use Closure;
use Core\Mod\Agentic\Mcp\Services\ToolDependencyService;
use Illuminate\Http\Request;
use RuntimeException;
use Symfony\Component\HttpFoundation\Response;

class ValidateToolDependencies
{
    public function __construct(
        protected ToolDependencyService $dependencyService,
    ) {}

    public function handle(Request $request, Closure $next): Response
    {
        if (! $this->isToolCallRequest($request)) {
            return $next($request);
        }

        $toolName = $this->extractToolName($request);
        if ($toolName === null || $toolName === '') {
            return $next($request);
        }

        $arguments = $this->extractArguments($request);
        $context = $this->extractContext($request);
        $sessionId = $this->extractSessionId($request, $context);

        try {
            $missing = $this->dependencyService->missing($toolName, $context, $arguments, $sessionId);
        } catch (RuntimeException $exception) {
            return response()->json([
                'error' => 'dependency_validation_failed',
                'message' => $exception->getMessage(),
                'tool' => $toolName,
            ], 409);
        }

        if ($missing !== []) {
            return response()->json([
                'error' => 'dependency_not_met',
                'message' => $missing[0]['message'] ?? 'Tool dependencies are not met.',
                'tool' => $toolName,
                'missing_dependencies' => $missing,
            ], 409);
        }

        $response = $next($request);

        if ($sessionId !== null && $response->getStatusCode() < 400) {
            $this->dependencyService->recordToolCall($sessionId, $toolName, $arguments);
        }

        return $response;
    }

    protected function isToolCallRequest(Request $request): bool
    {
        $method = (string) $request->input('method', '');

        return $request->isMethod('POST')
            && (
                $request->is('*/tools/call')
                || $method === 'tools/call'
                || $method === 'mcp.tools/call'
            );
    }

    protected function extractToolName(Request $request): ?string
    {
        $toolName = $request->input('params.name')
            ?? $request->input('params.tool')
            ?? $request->input('tool')
            ?? $request->input('name');

        return is_string($toolName) && $toolName !== '' ? $toolName : null;
    }

    /**
     * @return array<string, mixed>
     */
    protected function extractArguments(Request $request): array
    {
        $arguments = $request->input('params.arguments', $request->input('arguments', []));

        return is_array($arguments) ? $arguments : [];
    }

    /**
     * @return array<string, mixed>
     */
    protected function extractContext(Request $request): array
    {
        $context = [];
        $existingContext = $request->attributes->get('mcp_workspace_context');
        if (is_array($existingContext)) {
            $context = $existingContext;
        }

        $workspaceId = $request->attributes->get('workspace_id');
        if (is_int($workspaceId) || is_string($workspaceId)) {
            $context['workspace_id'] = $workspaceId;
        }

        $requestContext = $request->input('params.context', $request->input('context', []));
        if (is_array($requestContext)) {
            $context = array_merge($context, $requestContext);
        }

        return $context;
    }

    /**
     * @param  array<string, mixed>  $context
     */
    protected function extractSessionId(Request $request, array $context): ?string
    {
        $sessionId = $request->input('params.arguments.session_id')
            ?? $request->input('arguments.session_id')
            ?? $request->input('params.context.session_id')
            ?? $request->input('context.session_id')
            ?? $request->input('session_id')
            ?? $request->header('X-MCP-Session-ID')
            ?? $request->attributes->get('session_id')
            ?? ($context['session_id'] ?? null);

        return is_string($sessionId) && $sessionId !== '' ? $sessionId : null;
    }
}
