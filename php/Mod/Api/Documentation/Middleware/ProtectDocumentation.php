<?php

// SPDX-License-Identifier: EUPL-1.2

declare(strict_types=1);

namespace Core\Mod\Agentic\Mod\Api\Documentation\Middleware;

use Closure;
use Illuminate\Http\Request;
use Symfony\Component\HttpFoundation\Response;

class ProtectDocumentation
{
    public function handle(Request $request, Closure $next): Response
    {
        if (! config('api-docs.enabled', true)) {
            abort(404);
        }

        $config = config('api-docs.access', []);
        $publicEnvironments = $config['public_environments'] ?? ['local', 'testing', 'staging'];

        if (in_array(app()->environment(), $publicEnvironments, true)) {
            return $next($request);
        }

        $ipWhitelist = $config['ip_whitelist'] ?? [];
        if ($ipWhitelist !== []) {
            if (! in_array($request->ip(), $ipWhitelist, true)) {
                abort(403, 'Access denied.');
            }

            return $next($request);
        }

        if (($config['require_auth'] ?? false) && ! $request->user()) {
            abort(403, 'Documentation access requires authentication.');
        }

        return $next($request);
    }
}
