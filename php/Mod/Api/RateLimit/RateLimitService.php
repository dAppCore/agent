<?php

// SPDX-License-Identifier: EUPL-1.2

declare(strict_types=1);

namespace Core\Mod\Agentic\Mod\Api\RateLimit;

use Carbon\Carbon;
use Illuminate\Contracts\Cache\Repository as CacheRepository;
use InvalidArgumentException;

class RateLimitService
{
    protected const CACHE_PREFIX = 'rate_limit:';

    public function __construct(
        protected CacheRepository $cache,
    ) {}

    public function checkLimit(mixed $apiKey = null, string $endpoint = 'global', ?RateLimit $rateLimit = null): RateLimitResult
    {
        $rateLimit ??= $this->defaultRateLimit();

        return $this->check(
            key: $this->resolveLimitKey($apiKey, $endpoint, $rateLimit),
            limit: $rateLimit->limit,
            window: $rateLimit->window,
            burst: $rateLimit->burst,
        );
    }

    public function recordHit(mixed $apiKey = null, string $endpoint = 'global', ?RateLimit $rateLimit = null): RateLimitResult
    {
        $rateLimit ??= $this->defaultRateLimit();

        return $this->hit(
            key: $this->resolveLimitKey($apiKey, $endpoint, $rateLimit),
            limit: $rateLimit->limit,
            window: $rateLimit->window,
            burst: $rateLimit->burst,
        );
    }

    public function resetFor(mixed $apiKey = null, string $endpoint = 'global', ?RateLimit $rateLimit = null): void
    {
        $rateLimit ??= $this->defaultRateLimit();

        $this->reset($this->resolveLimitKey($apiKey, $endpoint, $rateLimit));
    }

    public function check(string $key, int $limit, int $window, float $burst = 1.0): RateLimitResult
    {
        $cacheKey = $this->getCacheKey($key);
        $effectiveLimit = $this->effectiveLimit($limit, $burst);
        $this->guardWindow($window);

        $now = Carbon::now();
        $windowStart = $now->timestamp - $window;
        $hits = $this->getWindowHits($cacheKey, $windowStart);
        $currentCount = count($hits);
        $remaining = max(0, $effectiveLimit - $currentCount);
        $resetsAt = $this->calculateResetTime($hits, $window, $effectiveLimit);

        if ($currentCount >= $effectiveLimit) {
            $oldestHit = min($hits);
            $retryAfter = max(1, ($oldestHit + $window) - $now->timestamp);

            return RateLimitResult::denied($limit, $retryAfter, $resetsAt);
        }

        return RateLimitResult::allowed($limit, $remaining, $resetsAt);
    }

    public function hit(string $key, int $limit, int $window, float $burst = 1.0): RateLimitResult
    {
        $cacheKey = $this->getCacheKey($key);
        $effectiveLimit = $this->effectiveLimit($limit, $burst);
        $this->guardWindow($window);

        $now = Carbon::now();
        $windowStart = $now->timestamp - $window;
        $hits = $this->getWindowHits($cacheKey, $windowStart);

        if (count($hits) >= $effectiveLimit) {
            $oldestHit = min($hits);
            $retryAfter = max(1, ($oldestHit + $window) - $now->timestamp);

            return RateLimitResult::denied(
                $limit,
                $retryAfter,
                $this->calculateResetTime($hits, $window, $effectiveLimit),
            );
        }

        $hits[] = $now->timestamp;
        $this->storeWindowHits($cacheKey, $hits, $window);

        return RateLimitResult::allowed(
            $limit,
            max(0, $effectiveLimit - count($hits)),
            $this->calculateResetTime($hits, $window, $effectiveLimit),
        );
    }

    public function remaining(string $key, int $limit, int $window, float $burst = 1.0): int
    {
        $this->guardWindow($window);

        return max(
            0,
            $this->effectiveLimit($limit, $burst)
            - count($this->getWindowHits($this->getCacheKey($key), Carbon::now()->timestamp - $window)),
        );
    }

    public function reset(string $key): void
    {
        $this->cache->forget($this->getCacheKey($key));
    }

    public function attempts(string $key, int $window): int
    {
        $this->guardWindow($window);

        return count($this->getWindowHits($this->getCacheKey($key), Carbon::now()->timestamp - $window));
    }

    public function buildEndpointKey(string $identifier, string $endpoint): string
    {
        return "endpoint:{$identifier}:{$endpoint}";
    }

    public function buildWorkspaceKey(int $workspaceId, ?string $suffix = null): string
    {
        return $suffix === null
            ? "workspace:{$workspaceId}"
            : "workspace:{$workspaceId}:{$suffix}";
    }

    public function buildApiKeyKey(int|string $apiKeyId, ?string $suffix = null): string
    {
        return $suffix === null
            ? "api_key:{$apiKeyId}"
            : "api_key:{$apiKeyId}:{$suffix}";
    }

    public function buildIpKey(string $ip, ?string $suffix = null): string
    {
        return $suffix === null
            ? "ip:{$ip}"
            : "ip:{$ip}:{$suffix}";
    }

    /**
     * @return array<int, int>
     */
    protected function getWindowHits(string $cacheKey, int $windowStart): array
    {
        $hits = $this->cache->get($cacheKey, []);

        if (! is_array($hits)) {
            return [];
        }

        $windowHits = [];

        foreach ($hits as $timestamp) {
            if (! is_int($timestamp) && ! is_float($timestamp) && ! is_string($timestamp)) {
                continue;
            }

            $timestamp = (int) $timestamp;

            if ($timestamp >= $windowStart) {
                $windowHits[] = $timestamp;
            }
        }

        return $windowHits;
    }

    /**
     * @param  array<int, int>  $hits
     */
    protected function storeWindowHits(string $cacheKey, array $hits, int $window): void
    {
        $this->cache->put($cacheKey, $hits, $window + 60);
    }

    /**
     * @param  array<int, int>  $hits
     */
    protected function calculateResetTime(array $hits, int $window, int $limit): Carbon
    {
        if ($hits === [] || count($hits) < $limit) {
            return Carbon::now()->addSeconds($window);
        }

        return Carbon::createFromTimestamp(min($hits) + $window);
    }

    protected function getCacheKey(string $key): string
    {
        return self::CACHE_PREFIX.$key;
    }

    protected function resolveLimitKey(mixed $apiKey, string $endpoint, RateLimit $rateLimit): string
    {
        if ($rateLimit->key !== '') {
            return $rateLimit->key;
        }

        return $this->buildEndpointKey(
            $this->normaliseApiKey($apiKey),
            $this->normaliseEndpoint($endpoint),
        );
    }

    protected function defaultRateLimit(): RateLimit
    {
        $config = config('api.rate_limiting', []);

        if (! is_array($config) || $config === []) {
            return new RateLimit(limit: 60, window: 60, burst: 1.0);
        }

        return new RateLimit(
            limit: (int) ($config['default_limit'] ?? $config['limit'] ?? 60),
            window: (int) ($config['default_window'] ?? $config['window'] ?? 60),
            burst: (float) ($config['default_burst'] ?? $config['burst'] ?? 1.0),
        );
    }

    protected function effectiveLimit(int $limit, float $burst): int
    {
        if ($limit < 1) {
            throw new InvalidArgumentException('Rate limit must be at least 1.');
        }

        if ($burst <= 0) {
            throw new InvalidArgumentException('Rate limit burst must be greater than zero.');
        }

        return max(1, (int) floor($limit * $burst));
    }

    protected function guardWindow(int $window): void
    {
        if ($window < 1) {
            throw new InvalidArgumentException('Rate limit window must be at least 1 second.');
        }
    }

    protected function normaliseApiKey(mixed $apiKey): string
    {
        if ($apiKey === null || $apiKey === '') {
            return 'api_key:anonymous';
        }

        if (is_int($apiKey) || is_string($apiKey)) {
            $identifier = (string) $apiKey;

            return str_contains($identifier, ':') ? $identifier : "api_key:{$identifier}";
        }

        if (is_object($apiKey)) {
            if (method_exists($apiKey, 'getKey')) {
                $key = $apiKey->getKey();

                if (is_int($key) || is_string($key)) {
                    return "api_key:{$key}";
                }
            }

            if (isset($apiKey->id) && (is_int($apiKey->id) || is_string($apiKey->id))) {
                return "api_key:{$apiKey->id}";
            }

            if (method_exists($apiKey, '__toString')) {
                return (string) $apiKey;
            }

            return 'api_key:object:'.spl_object_id($apiKey);
        }

        return 'api_key:'.md5(serialize($apiKey));
    }

    protected function normaliseEndpoint(string $endpoint): string
    {
        $endpoint = trim($endpoint);

        return $endpoint === '' ? 'global' : $endpoint;
    }
}
