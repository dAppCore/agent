<?php

// SPDX-License-Identifier: EUPL-1.2

declare(strict_types=1);

namespace Core\Mod\Agentic\Pipeline;

final readonly class Reactions
{
    public function __construct(
        public int $plusOne = 0,
        public int $minusOne = 0,
        public int $laugh = 0,
        public int $hooray = 0,
        public int $confused = 0,
        public int $heart = 0,
        public int $rocket = 0,
        public int $eyes = 0,
    ) {}

    /**
     * @return array<string, int>
     */
    public function toArray(): array
    {
        return [
            '+1' => $this->plusOne,
            '-1' => $this->minusOne,
            'laugh' => $this->laugh,
            'hooray' => $this->hooray,
            'confused' => $this->confused,
            'heart' => $this->heart,
            'rocket' => $this->rocket,
            'eyes' => $this->eyes,
        ];
    }
}
