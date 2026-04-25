<?php

// SPDX-License-Identifier: EUPL-1.2

declare(strict_types=1);

namespace Core\Mcp\Services;

use Illuminate\Support\Facades\Cache;

final class ToolRateLimiter
{
    protected const CACHE_PREFIX = 'mcp_rate_limit:';

    public function check(string $identifier, string $toolName): array
    {
        if (! config('mcp.rate_limiting.enabled', true)) {
            return ['limited' => false, 'remaining' => PHP_INT_MAX, 'retry_after' => null];
        }

        $limit = $this->limitForTool($toolName);
        $cacheKey = $this->cacheKey($identifier, $toolName);
        $current = (int) Cache::get($cacheKey, 0);
        $decaySeconds = (int) config('mcp.rate_limiting.decay_seconds', 60);

        if ($current >= $limit) {
            $ttl = $this->ttl($cacheKey, $decaySeconds);

            return [
                'limited' => true,
                'remaining' => 0,
                'retry_after' => $ttl > 0 ? $ttl : $decaySeconds,
            ];
        }

        return [
            'limited' => false,
            'remaining' => max($limit - $current - 1, 0),
            'retry_after' => null,
        ];
    }

    public function hit(string $identifier, string $toolName): void
    {
        if (! config('mcp.rate_limiting.enabled', true)) {
            return;
        }

        $cacheKey = $this->cacheKey($identifier, $toolName);
        $current = (int) Cache::get($cacheKey, 0);
        $decaySeconds = (int) config('mcp.rate_limiting.decay_seconds', 60);

        if ($current === 0) {
            Cache::put($cacheKey, 1, $decaySeconds);

            return;
        }

        Cache::increment($cacheKey);
    }

    public function clear(string $identifier, ?string $toolName = null): void
    {
        if ($toolName !== null) {
            Cache::forget($this->cacheKey($identifier, $toolName));

            return;
        }

        foreach (array_keys((array) config('mcp.rate_limiting.per_tool', [])) as $configuredTool) {
            Cache::forget($this->cacheKey($identifier, (string) $configuredTool));
        }

        Cache::forget($this->cacheKey($identifier, '*'));
    }

    public function getStatus(string $identifier, string $toolName): array
    {
        $limit = $this->limitForTool($toolName);
        $cacheKey = $this->cacheKey($identifier, $toolName);
        $current = (int) Cache::get($cacheKey, 0);
        $ttl = $this->ttl($cacheKey, (int) config('mcp.rate_limiting.decay_seconds', 60));

        return [
            'limit' => $limit,
            'remaining' => max($limit - $current, 0),
            'reset_at' => $ttl > 0 ? now()->addSeconds($ttl)->toIso8601String() : null,
        ];
    }

    protected function limitForTool(string $toolName): int
    {
        $perTool = (array) config('mcp.rate_limiting.per_tool', []);

        if (array_key_exists($toolName, $perTool)) {
            return (int) $perTool[$toolName];
        }

        return (int) config('mcp.rate_limiting.calls_per_minute', 60);
    }

    protected function cacheKey(string $identifier, string $toolName): string
    {
        return self::CACHE_PREFIX.$identifier.':'.$toolName;
    }

    protected function ttl(string $cacheKey, int $default): int
    {
        try {
            $ttl = Cache::ttl($cacheKey);

            return is_int($ttl) ? $ttl : $default;
        } catch (\Throwable) {
            return $default;
        }
    }
}
