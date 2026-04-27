<?php

// SPDX-License-Identifier: EUPL-1.2

declare(strict_types=1);

namespace Core\Mod\Agentic\Mod\Api\Middleware;

use Closure;
use Core\Mod\Agentic\Mod\Api\Models\ApiKey;
use Illuminate\Http\Request;
use Symfony\Component\HttpFoundation\Response;

class AuthenticateApiKey
{
    public function handle(Request $request, Closure $next, ?string $scope = null): Response
    {
        $token = $request->bearerToken();

        if ($token === null || $token === '') {
            return $this->unauthorised('API key required. Use Authorization: Bearer <api_key>');
        }

        if (str_starts_with($token, 'hk_')) {
            return $this->authenticateApiKey($request, $next, $token, $scope);
        }

        return $this->authenticateSanctum($request, $next);
    }

    protected function authenticateApiKey(Request $request, Closure $next, string $token, ?string $scope): Response
    {
        $apiKey = ApiKey::findByPlainKey($token);

        if (! $apiKey instanceof ApiKey) {
            return $this->unauthorised('Invalid API key');
        }

        if ($apiKey->isExpired()) {
            return $this->unauthorised('API key has expired');
        }

        if ($apiKey->hasIpRestrictions() && ! in_array((string) $request->ip(), $apiKey->getAllowedIps(), true)) {
            return $this->forbidden('IP address not allowed for this API key');
        }

        if ($scope !== null && ! $apiKey->hasScope($scope)) {
            return $this->forbidden("API key missing required scope: {$scope}");
        }

        $apiKey->recordUsage();

        $request->setUserResolver(fn () => $apiKey->user);
        $request->attributes->set('api_key', $apiKey);
        $request->attributes->set('workspace', $apiKey->workspace);
        $request->attributes->set('workspace_id', $apiKey->workspace_id);
        $request->attributes->set('auth_type', 'api_key');

        return $next($request);
    }

    protected function authenticateSanctum(Request $request, Closure $next): Response
    {
        if (! $request->user()) {
            $guard = auth('sanctum');

            if (! $guard->check()) {
                return $this->unauthorised('Invalid authentication token');
            }

            $request->setUserResolver(fn () => $guard->user());
        }

        $request->attributes->set('auth_type', 'sanctum');

        return $next($request);
    }

    protected function unauthorised(string $message): Response
    {
        return response()->json([
            'error' => 'unauthorised',
            'message' => $message,
        ], 401);
    }

    protected function forbidden(string $message): Response
    {
        return response()->json([
            'error' => 'forbidden',
            'message' => $message,
        ], 403);
    }
}
