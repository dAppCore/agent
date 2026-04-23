<?php

// SPDX-License-Identifier: EUPL-1.2

declare(strict_types=1);

namespace Core\Mod\Agentic\Pipeline;

final readonly class EpicChild
{
    public function __construct(
        public int $issueId,
        public string $state,
        public bool $checkedBool,
        public ?int $linkedPrNumberOrNull,
    ) {}

    /**
     * @return array<string, mixed>
     */
    public function toArray(): array
    {
        return [
            'issue_id' => $this->issueId,
            'state' => $this->state,
            'checked_bool' => $this->checkedBool,
            'linked_pr_number_or_null' => $this->linkedPrNumberOrNull,
        ];
    }
}
