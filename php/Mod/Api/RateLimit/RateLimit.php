<?php

// SPDX-License-Identifier: EUPL-1.2

declare(strict_types=1);

namespace Core\Mod\Agentic\Mod\Api\RateLimit;

use Attribute;

#[Attribute(Attribute::TARGET_CLASS | Attribute::TARGET_METHOD)]
readonly class RateLimit
{
    public string $key;

    public int $limit;

    public int $window;

    public float $burst;

    public function __construct(
        string|int $key = '',
        ?int $limit = null,
        int $window = 60,
        float $burst = 1.0,
    ) {
        if (is_int($key) && $limit === null) {
            $this->key = '';
            $this->limit = $key;
        } else {
            $this->key = (string) $key;
            $this->limit = $limit ?? 60;
        }

        $this->window = $window;
        $this->burst = $burst;
    }
}
