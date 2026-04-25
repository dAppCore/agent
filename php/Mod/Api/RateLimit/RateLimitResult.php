<?php

// SPDX-License-Identifier: EUPL-1.2

declare(strict_types=1);

namespace Core\Mod\Agentic\Mod\Api\RateLimit;

use Carbon\Carbon;

readonly class RateLimitResult
{
    public Carbon $resetsAt;

    public function __construct(
        public bool $allowed,
        public int $limit,
        public int $remaining,
        public ?int $retryAfter,
        public Carbon $resetAt,
    ) {
        $this->resetsAt = $resetAt;
    }

    public static function allowed(int $limit, int $remaining, Carbon $resetAt): self
    {
        return new self(true, $limit, $remaining, null, $resetAt);
    }

    public static function denied(int $limit, int $retryAfter, Carbon $resetAt): self
    {
        return new self(false, $limit, 0, $retryAfter, $resetAt);
    }

    /**
     * @return array<string, int>
     */
    public function headers(): array
    {
        $headers = [
            'X-RateLimit-Limit' => $this->limit,
            'X-RateLimit-Remaining' => $this->remaining,
            'X-RateLimit-Reset' => $this->resetAt->timestamp,
        ];

        if (! $this->allowed && $this->retryAfter !== null) {
            $headers['Retry-After'] = $this->retryAfter;
        }

        return $headers;
    }
}
