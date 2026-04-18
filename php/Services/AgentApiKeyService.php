<?php

declare(strict_types=1);

namespace Core\Mod\Agentic\Services;

use Core\Mod\Agentic\Models\AgentApiKey;
use Core\Tenant\Models\Workspace;
use Illuminate\Support\Facades\Cache;
use Illuminate\Support\Facades\Log;

/**
 * Agent API Key Service.
 *
 * Handles key creation, validation, rate limiting, and IP restrictions
 * for external agent access to Host Hub.
 */
class AgentApiKeyService
{
    protected ?IpRestrictionService $ipRestrictionService = null;

    /**
     * Get the IP restriction service instance.
     */
    protected function ipRestriction(): IpRestrictionService
    {
        if ($this->ipRestrictionService === null) {
            $this->ipRestrictionService = app(IpRestrictionService::class);
        }

        return $this->ipRestrictionService;
    }

    /**
     * Create a new API key.
     */
    public function create(
        Workspace|int $workspace,
        string $name,
        array $permissions = [],
        int $rateLimit = 100,
        ?\Carbon\Carbon $expiresAt = null
    ): AgentApiKey {
        return AgentApiKey::generate(
            $workspace,
            $name,
            $permissions,
            $rateLimit,
            $expiresAt
        );
    }

    /**
     * Validate a key and return it if valid.
     */
    public function validate(string $plainKey): ?AgentApiKey
    {
        $key = AgentApiKey::findByKey($plainKey);

        if (! $key || ! $key->isActive()) {
            return null;
        }

        return $key;
    }

    /**
     * Check if a key has a specific permission.
     */
    public function checkPermission(AgentApiKey $key, string $permission): bool
    {
        if (! $key->isActive()) {
            return false;
        }

        return $key->hasPermission($permission);
    }

    /**
     * Check if a key has all required permissions.
     */
    public function checkPermissions(AgentApiKey $key, array $permissions): bool
    {
        if (! $key->isActive()) {
            return false;
        }

        return $key->hasAllPermissions($permissions);
    }

    /**
     * Record API key usage.
     *
     * @param  string|null  $clientIp  The client IP address to record
     */
    public function recordUsage(AgentApiKey $key, ?string $clientIp = null): void
    {
        $key->recordUsage();

        // Record the client IP if provided
        if ($clientIp !== null) {
            $key->recordLastUsedIp($clientIp);
        }

        // Increment rate limit counter in cache using atomic add
        // Cache::add() only sets the key if it doesn't exist, avoiding race condition
        $cacheKey = $this->getRateLimitCacheKey($key);
        $ttl = 60; // 60 seconds

        // Try to add with initial value of 1 and TTL
        // If key already exists, this returns false and we increment instead
        if (! Cache::add($cacheKey, 1, $ttl)) {
            Cache::increment($cacheKey);
        }
    }

    /**
     * Check if a key is rate limited.
     */
    public function isRateLimited(AgentApiKey $key): bool
    {
        $cacheKey = $this->getRateLimitCacheKey($key);
        $currentCalls = (int) Cache::get($cacheKey, 0);

        return $currentCalls >= $key->rate_limit;
    }

    /**
     * Get current rate limit status.
     */
    public function getRateLimitStatus(AgentApiKey $key): array
    {
        $cacheKey = $this->getRateLimitCacheKey($key);
        $currentCalls = (int) Cache::get($cacheKey, 0);
        $remaining = max(0, $key->rate_limit - $currentCalls);

        // Get TTL (remaining seconds until reset)
        $ttl = Cache::getStore() instanceof \Illuminate\Cache\RedisStore
            ? Cache::connection()->ttl($cacheKey)
            : 60;

        return [
            'limit' => $key->rate_limit,
            'remaining' => $remaining,
            'reset_in_seconds' => max(0, $ttl),
            'used' => $currentCalls,
        ];
    }

    /**
     * Revoke a key immediately.
     */
    public function revoke(AgentApiKey $key): void
    {
        $key->revoke();

        // Clear rate limit cache
        Cache::forget($this->getRateLimitCacheKey($key));

        // Clear permitted tools cache so the revoked key can no longer access tools
        app(AgentToolRegistry::class)->flushCacheForApiKey($key->id);
    }

    /**
     * Update key permissions.
     */
    public function updatePermissions(AgentApiKey $key, array $permissions): void
    {
        $key->updatePermissions($permissions);

        // Invalidate cached tool list so the new permissions take effect immediately
        app(AgentToolRegistry::class)->flushCacheForApiKey($key->id);
    }

    /**
     * Update key rate limit.
     */
    public function updateRateLimit(AgentApiKey $key, int $rateLimit): void
    {
        $key->updateRateLimit($rateLimit);
    }

    /**
     * Update IP restriction settings for a key.
     *
     * @param  array<string>  $whitelist
     */
    public function updateIpRestrictions(AgentApiKey $key, bool $enabled, array $whitelist = []): void
    {
        $key->update([
            'ip_restriction_enabled' => $enabled,
            'ip_whitelist' => $whitelist,
        ]);
    }

    /**
     * Enable IP restrictions with a whitelist.
     *
     * @param  array<string>  $whitelist
     */
    public function enableIpRestrictions(AgentApiKey $key, array $whitelist): void
    {
        $key->enableIpRestriction();
        $key->updateIpWhitelist($whitelist);
    }

    /**
     * Disable IP restrictions.
     */
    public function disableIpRestrictions(AgentApiKey $key): void
    {
        $key->disableIpRestriction();
    }

    /**
     * Parse and validate IP whitelist input.
     *
     * @return array{entries: array<string>, errors: array<string>}
     */
    public function parseIpWhitelistInput(string $input): array
    {
        return $this->ipRestriction()->parseWhitelistInput($input);
    }

    /**
     * Check if an IP is allowed for a key.
     */
    public function isIpAllowed(AgentApiKey $key, string $ip): bool
    {
        return $this->ipRestriction()->validateIp($key, $ip);
    }

    /**
     * Extend key expiration.
     */
    public function extendExpiry(AgentApiKey $key, \Carbon\Carbon $expiresAt): void
    {
        $key->extendExpiry($expiresAt);
    }

    /**
     * Remove key expiration (make permanent).
     */
    public function removeExpiry(AgentApiKey $key): void
    {
        $key->removeExpiry();
    }

    /**
     * Get all active keys for a workspace.
     */
    public function getActiveKeysForWorkspace(Workspace|int $workspace): \Illuminate\Database\Eloquent\Collection
    {
        return AgentApiKey::active()
            ->forWorkspace($workspace)
            ->orderBy('name')
            ->get();
    }

    /**
     * Get all keys (including inactive) for a workspace.
     */
    public function getAllKeysForWorkspace(Workspace|int $workspace): \Illuminate\Database\Eloquent\Collection
    {
        return AgentApiKey::forWorkspace($workspace)
            ->orderByDesc('created_at')
            ->get();
    }

    /**
     * Validate a key and check permission in one call.
     * Returns the key if valid with permission, null otherwise.
     */
    public function validateWithPermission(string $plainKey, string $permission): ?AgentApiKey
    {
        $key = $this->validate($plainKey);

        if (! $key) {
            return null;
        }

        if (! $this->checkPermission($key, $permission)) {
            return null;
        }

        if ($this->isRateLimited($key)) {
            return null;
        }

        return $key;
    }

    /**
     * Full authentication flow for API requests.
     * Returns array with key and status info, or error.
     *
     * @param  string|null  $clientIp  The client IP address for IP restriction checking
     */
    public function authenticate(string $plainKey, string $requiredPermission, ?string $clientIp = null): array
    {
        $key = AgentApiKey::findByKey($plainKey);

        if (! $key) {
            return [
                'success' => false,
                'error' => 'invalid_key',
                'message' => 'Invalid API key',
            ];
        }

        if ($key->isRevoked()) {
            return [
                'success' => false,
                'error' => 'key_revoked',
                'message' => 'API key has been revoked',
            ];
        }

        if ($key->isExpired()) {
            return [
                'success' => false,
                'error' => 'key_expired',
                'message' => 'API key has expired',
            ];
        }

        // Check IP restrictions
        if ($clientIp !== null && $key->ip_restriction_enabled) {
            if (! $this->ipRestriction()->validateIp($key, $clientIp)) {
                // Log blocked attempt
                Log::warning('API key IP restriction blocked', [
                    'key_id' => $key->id,
                    'key_name' => $key->name,
                    'workspace_id' => $key->workspace_id,
                    'blocked_ip' => $clientIp,
                    'whitelist_count' => $key->getIpWhitelistCount(),
                ]);

                return [
                    'success' => false,
                    'error' => 'ip_not_allowed',
                    'message' => 'Request IP is not in the allowed whitelist',
                    'client_ip' => $clientIp,
                ];
            }
        }

        if (! $key->hasPermission($requiredPermission)) {
            return [
                'success' => false,
                'error' => 'permission_denied',
                'message' => "Missing required permission: {$requiredPermission}",
            ];
        }

        if ($this->isRateLimited($key)) {
            $status = $this->getRateLimitStatus($key);

            return [
                'success' => false,
                'error' => 'rate_limited',
                'message' => 'Rate limit exceeded',
                'rate_limit' => $status,
            ];
        }

        // Record successful usage with IP
        $this->recordUsage($key, $clientIp);

        return [
            'success' => true,
            'key' => $key,
            'workspace_id' => $key->workspace_id,
            'rate_limit' => $this->getRateLimitStatus($key),
            'client_ip' => $clientIp,
        ];
    }

    /**
     * Get cache key for rate limiting.
     */
    private function getRateLimitCacheKey(AgentApiKey $key): string
    {
        return "agent_api_key_rate:{$key->id}";
    }
}
