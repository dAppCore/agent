<?php

// SPDX-License-Identifier: EUPL-1.2

declare(strict_types=1);

namespace Core\Mod\Agentic\Pipeline;

final readonly class PRMeta
{
    /**
     * @param  array<int, array{name: string, conclusion: string|null, status: string|null}>  $checkStatuses
     */
    public function __construct(
        public string $state,
        public string $mergeability,
        public ?string $headSha,
        public ?string $headDate,
        public ?string $baseBranch,
        public ?string $headBranch,
        public array $checkStatuses,
        public int $reviewThreadsTotal,
        public int $reviewThreadsResolved,
        public bool $hasEyesReaction,
    ) {}

    /**
     * @return array<string, mixed>
     */
    public function toArray(): array
    {
        return [
            'state' => $this->state,
            'mergeability' => $this->mergeability,
            'head_sha' => $this->headSha,
            'head_date' => $this->headDate,
            'base_branch' => $this->baseBranch,
            'head_branch' => $this->headBranch,
            'check_statuses' => $this->checkStatuses,
            'review_threads_total' => $this->reviewThreadsTotal,
            'review_threads_resolved' => $this->reviewThreadsResolved,
            'has_eyes_reaction' => $this->hasEyesReaction,
        ];
    }
}
