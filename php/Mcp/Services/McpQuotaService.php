<?php

// SPDX-License-Identifier: EUPL-1.2

declare(strict_types=1);

namespace Core\Mod\Agentic\Mcp\Services;

use Carbon\CarbonImmutable;
use Core\Mod\Agentic\Mcp\Data\QuotaResult;
use Illuminate\Support\Facades\Cache;
use InvalidArgumentException;

final class McpQuotaService
{
    public function checkQuota(int|string $workspaceId, ?string $period = null): QuotaResult
    {
        $resolvedPeriod = $this->resolvePeriod($period);
        $limit = $this->limitFor($workspaceId, $resolvedPeriod);
        $used = (int) Cache::get($this->usageKey($workspaceId, $resolvedPeriod), 0);
        $resetAt = $this->resetAt($resolvedPeriod);

        return new QuotaResult(
            workspaceId: (string) $workspaceId,
            period: $resolvedPeriod,
            limit: $limit,
            used: $used,
            remaining: max($limit - $used, 0),
            resetAt: $resetAt,
            exceeded: $used >= $limit,
        );
    }

    public function consume(int|string $workspaceId, int $units = 1, ?string $period = null): QuotaResult
    {
        if ($units < 1) {
            throw new InvalidArgumentException('Quota consumption units must be at least 1.');
        }

        $resolvedPeriod = $this->resolvePeriod($period);
        $usageKey = $this->usageKey($workspaceId, $resolvedPeriod);
        $resetAt = $this->resetAt($resolvedPeriod);
        $updatedUsage = (int) Cache::get($usageKey, 0) + $units;

        Cache::put($usageKey, $updatedUsage, $resetAt);

        return $this->checkQuota($workspaceId, $resolvedPeriod);
    }

    public function reset(int|string $workspaceId, ?string $period = null): QuotaResult
    {
        $resolvedPeriod = $this->resolvePeriod($period);

        Cache::forget($this->usageKey($workspaceId, $resolvedPeriod));

        return $this->checkQuota($workspaceId, $resolvedPeriod);
    }

    public function isExceeded(int|string $workspaceId, string $period = 'minute'): bool
    {
        return $this->checkQuota($workspaceId, $period)->exceeded;
    }

    public function increment(int|string $workspaceId, string $period = 'minute'): void
    {
        $this->consume($workspaceId, 1, $period);
    }

    public function currentUsage(int|string $workspaceId, string $period = 'minute'): int
    {
        return $this->checkQuota($workspaceId, $period)->used;
    }

    public function setQuota(int|string $workspaceId, int $limit, string $period = 'minute'): QuotaResult
    {
        if ($limit < 1) {
            throw new InvalidArgumentException('Quota limit must be at least 1.');
        }

        $resolvedPeriod = $this->resolvePeriod($period);

        Cache::forever($this->limitKey($workspaceId, $resolvedPeriod), $limit);

        return $this->checkQuota($workspaceId, $resolvedPeriod);
    }

    private function limitFor(int|string $workspaceId, string $period): int
    {
        return (int) Cache::get(
            $this->limitKey($workspaceId, $period),
            (int) config('mcp.quota_limit', 1000),
        );
    }

    private function resolvePeriod(?string $period): string
    {
        $resolved = (string) ($period ?? config('mcp.quota_period', 'minute'));

        if (! in_array($resolved, ['minute', 'hour', 'day'], true)) {
            throw new InvalidArgumentException(sprintf(
                'Unsupported quota period [%s].',
                $resolved,
            ));
        }

        return $resolved;
    }

    private function usageKey(int|string $workspaceId, string $period): string
    {
        $window = match ($period) {
            'minute' => $this->windowStart($period)->format('YmdHi'),
            'hour' => $this->windowStart($period)->format('YmdH'),
            'day' => $this->windowStart($period)->format('Ymd'),
        };

        return sprintf('mcp:quota:usage:%s:%s:%s', $period, $workspaceId, $window);
    }

    private function limitKey(int|string $workspaceId, string $period): string
    {
        return sprintf('mcp:quota:limit:%s:%s', $period, $workspaceId);
    }

    private function windowStart(string $period): CarbonImmutable
    {
        $now = CarbonImmutable::now();

        return match ($period) {
            'minute' => $now->startOfMinute(),
            'hour' => $now->startOfHour(),
            'day' => $now->startOfDay(),
        };
    }

    private function resetAt(string $period): CarbonImmutable
    {
        return match ($period) {
            'minute' => $this->windowStart($period)->addMinute(),
            'hour' => $this->windowStart($period)->addHour(),
            'day' => $this->windowStart($period)->addDay(),
        };
    }
}
