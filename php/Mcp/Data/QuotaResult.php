<?php

// SPDX-License-Identifier: EUPL-1.2

declare(strict_types=1);

namespace Core\Mod\Agentic\Mcp\Data;

use Carbon\CarbonImmutable;

final readonly class QuotaResult
{
    public function __construct(
        public string $workspaceId,
        public string $period,
        public int $limit,
        public int $used,
        public int $remaining,
        public CarbonImmutable $resetAt,
        public bool $exceeded,
    ) {}

    public function toArray(): array
    {
        return [
            'workspace_id' => $this->workspaceId,
            'period' => $this->period,
            'limit' => $this->limit,
            'used' => $this->used,
            'remaining' => $this->remaining,
            'reset_at' => $this->resetAt->toIso8601String(),
            'exceeded' => $this->exceeded,
        ];
    }
}
