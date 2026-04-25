<?php

// SPDX-License-Identifier: EUPL-1.2

declare(strict_types=1);

namespace Core\Mod\Agentic\Website\Mcp\Middleware;

use Closure;
use Illuminate\Http\Request;
use Symfony\Component\HttpFoundation\Response;

class McpAuthenticate
{
    public function __construct(
        protected McpApiKeyAuth $apiKeyAuth,
        protected CheckMcpQuota $checkMcpQuota,
        protected ValidateWorkspaceContext $validateWorkspaceContext,
        protected ValidateToolDependencies $validateToolDependencies,
    ) {}

    public function handle(Request $request, Closure $next): Response
    {
        return $this->apiKeyAuth->handle(
            $request,
            fn (Request $authenticatedRequest): Response => $this->checkMcpQuota->handle(
                $authenticatedRequest,
                fn (Request $quotaCheckedRequest): Response => $this->validateWorkspaceContext->handle(
                    $quotaCheckedRequest,
                    fn (Request $workspaceValidatedRequest): Response => $this->validateToolDependencies->handle(
                        $workspaceValidatedRequest,
                        $next,
                    ),
                ),
            ),
        );
    }
}
