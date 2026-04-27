<?php

// SPDX-License-Identifier: EUPL-1.2

declare(strict_types=1);

namespace Core\Mod\Agentic\Pipeline;

final readonly class IssueState
{
    /**
     * @param  array<int, string>  $labels
     */
    public function __construct(
        public string $state,
        public string $title,
        public array $labels,
        public ?string $assignee,
    ) {}

    /**
     * @return array<string, mixed>
     */
    public function toArray(): array
    {
        return [
            'state' => $this->state,
            'title' => $this->title,
            'labels' => $this->labels,
            'assignee' => $this->assignee,
        ];
    }
}
