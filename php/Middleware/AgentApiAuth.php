<?php

declare(strict_types=1);

namespace Core\Mod\Agentic\Middleware;

use Closure;
use Core\Mod\Agentic\Models\AgentApiKey;
use Core\Mod\Agentic\Services\AgentApiKeyService;
use Illuminate\Http\Request;
use Symfony\Component\HttpFoundation\Response;

/**
 * Agent API Authentication Middleware.
 *
 * Authenticates API requests using Bearer tokens and validates:
 * - API key validity (exists, not revoked, not expired)
 * - Required permissions
 * - IP whitelist restrictions
 * - Rate limits
 *
 * Adds rate limit headers to responses and X-Client-IP for debugging.
 */
class AgentApiAuth
{
    public function __construct(
        protected AgentApiKeyService $keyService
    ) {}

    /**
     * Handle an incoming request.
     *
     * @param  string|array<string>  $permissions  Required permission(s)
     */
    public function handle(Request $request, Closure $next, string|array $permissions = []): Response
    {
        $token = $request->bearerToken();

        if (! $token) {
            return $this->unauthorised('API token required. Use Authorization: Bearer <token>');
        }

        // Normalise permissions to array
        if (is_string($permissions)) {
            $permissions = $permissions ? explode(',', $permissions) : [];
        }

        // Get client IP
        $clientIp = $request->ip();

        // Use the first permission for authenticate call, we'll check all below
        $primaryPermission = $permissions[0] ?? '';

        // Authenticate with IP check
        $result = $this->keyService->authenticate($token, $primaryPermission, $clientIp);

        if (! $result['success']) {
            return $this->handleAuthError($result, $clientIp);
        }

        /** @var AgentApiKey $key */
        $key = $result['key'];

        // Check all required permissions if multiple specified
        if (count($permissions) > 1) {
            foreach (array_slice($permissions, 1) as $permission) {
                if (! $key->hasPermission($permission)) {
                    return $this->forbidden("Missing required permission: {$permission}", $clientIp);
                }
            }
        }

        // Store API key in request for downstream use
        $request->attributes->set('agent_api_key', $key);
        $request->attributes->set('api_key', $key);
        $request->attributes->set('workspace_id', $key->workspace_id);
        $request->attributes->set('workspace', $key->workspace);

        /** @var Response $response */
        $response = $next($request);

        // Add rate limit headers
        $rateLimit = $result['rate_limit'] ?? [];
        if (! empty($rateLimit)) {
            $response->headers->set('X-RateLimit-Limit', (string) ($rateLimit['limit'] ?? 0));
            $response->headers->set('X-RateLimit-Remaining', (string) ($rateLimit['remaining'] ?? 0));
            $response->headers->set('X-RateLimit-Reset', (string) ($rateLimit['reset_in_seconds'] ?? 0));
        }

        // Add client IP header for debugging
        if ($clientIp) {
            $response->headers->set('X-Client-IP', $clientIp);
        }

        return $response;
    }

    /**
     * Handle authentication errors.
     */
    protected function handleAuthError(array $result, ?string $clientIp): Response
    {
        $error = $result['error'] ?? 'unknown_error';
        $message = $result['message'] ?? 'Authentication failed';

        return match ($error) {
            'invalid_key' => $this->unauthorised($message, $clientIp),
            'key_revoked' => $this->unauthorised($message, $clientIp),
            'key_expired' => $this->unauthorised($message, $clientIp),
            'ip_not_allowed' => $this->ipForbidden($message, $clientIp),
            'permission_denied' => $this->forbidden($message, $clientIp),
            'rate_limited' => $this->rateLimited($result, $clientIp),
            default => $this->unauthorised($message, $clientIp),
        };
    }

    /**
     * Return 401 Unauthorised response.
     */
    protected function unauthorised(string $message, ?string $clientIp = null): Response
    {
        return response()->json([
            'error' => 'unauthorised',
            'message' => $message,
        ], 401, $this->getBaseHeaders($clientIp));
    }

    /**
     * Return 403 Forbidden response.
     */
    protected function forbidden(string $message, ?string $clientIp = null): Response
    {
        return response()->json([
            'error' => 'forbidden',
            'message' => $message,
        ], 403, $this->getBaseHeaders($clientIp));
    }

    /**
     * Return 403 Forbidden response for IP restriction.
     */
    protected function ipForbidden(string $message, ?string $clientIp = null): Response
    {
        return response()->json([
            'error' => 'ip_not_allowed',
            'message' => $message,
            'your_ip' => $clientIp,
        ], 403, $this->getBaseHeaders($clientIp));
    }

    /**
     * Return 429 Too Many Requests response.
     */
    protected function rateLimited(array $result, ?string $clientIp = null): Response
    {
        $rateLimit = $result['rate_limit'] ?? [];

        $headers = array_merge($this->getBaseHeaders($clientIp), [
            'X-RateLimit-Limit' => (string) ($rateLimit['limit'] ?? 0),
            'X-RateLimit-Remaining' => '0',
            'X-RateLimit-Reset' => (string) ($rateLimit['reset_in_seconds'] ?? 60),
            'Retry-After' => (string) ($rateLimit['reset_in_seconds'] ?? 60),
        ]);

        return response()->json([
            'error' => 'rate_limited',
            'message' => $result['message'] ?? 'Rate limit exceeded',
            'rate_limit' => $rateLimit,
        ], 429, $headers);
    }

    /**
     * Get base headers to include in all responses.
     */
    protected function getBaseHeaders(?string $clientIp): array
    {
        $headers = [];

        if ($clientIp) {
            $headers['X-Client-IP'] = $clientIp;
        }

        return $headers;
    }
}
