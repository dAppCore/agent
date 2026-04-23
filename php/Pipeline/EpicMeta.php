<?php

// SPDX-License-Identifier: EUPL-1.2

declare(strict_types=1);

namespace Core\Mod\Agentic\Pipeline;

final readonly class EpicMeta
{
    /**
     * @param  array<int, EpicChild>  $children
     */
    public function __construct(
        public string $state,
        public array $children,
    ) {}

    /**
     * @return array<string, mixed>
     */
    public function toArray(): array
    {
        return [
            'state' => $this->state,
            'children' => array_map(
                static fn (EpicChild $child): array => $child->toArray(),
                $this->children,
            ),
        ];
    }
}
