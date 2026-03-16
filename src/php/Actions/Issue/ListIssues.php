<?php

declare(strict_types=1);

namespace Core\Mod\Agentic\Actions\Issue;

use Core\Actions\Action;
use Core\Mod\Agentic\Models\Issue;
use Core\Mod\Agentic\Models\Sprint;
use Illuminate\Support\Collection;

/**
 * List issues for a workspace with optional filtering.
 *
 * Usage:
 *   $issues = ListIssues::run(1);
 *   $issues = ListIssues::run(1, status: 'open', type: 'bug');
 */
class ListIssues
{
    use Action;

    /**
     * @return Collection<int, Issue>
     */
    public function handle(
        int $workspaceId,
        ?string $status = null,
        ?string $type = null,
        ?string $priority = null,
        ?string $sprintSlug = null,
        ?string $label = null,
        bool $includeClosed = false,
    ): Collection {
        $validStatuses = [Issue::STATUS_OPEN, Issue::STATUS_IN_PROGRESS, Issue::STATUS_REVIEW, Issue::STATUS_CLOSED];
        if ($status !== null && ! in_array($status, $validStatuses, true)) {
            throw new \InvalidArgumentException(
                sprintf('status must be one of: %s', implode(', ', $validStatuses))
            );
        }

        $validTypes = [Issue::TYPE_BUG, Issue::TYPE_FEATURE, Issue::TYPE_TASK, Issue::TYPE_IMPROVEMENT];
        if ($type !== null && ! in_array($type, $validTypes, true)) {
            throw new \InvalidArgumentException(
                sprintf('type must be one of: %s', implode(', ', $validTypes))
            );
        }

        $validPriorities = [Issue::PRIORITY_LOW, Issue::PRIORITY_NORMAL, Issue::PRIORITY_HIGH, Issue::PRIORITY_URGENT];
        if ($priority !== null && ! in_array($priority, $validPriorities, true)) {
            throw new \InvalidArgumentException(
                sprintf('priority must be one of: %s', implode(', ', $validPriorities))
            );
        }

        $query = Issue::with('sprint')
            ->forWorkspace($workspaceId)
            ->orderByPriority()
            ->orderBy('updated_at', 'desc');

        if (! $includeClosed && $status !== Issue::STATUS_CLOSED) {
            $query->notClosed();
        }

        if ($status !== null) {
            $query->where('status', $status);
        }

        if ($type !== null) {
            $query->ofType($type);
        }

        if ($priority !== null) {
            $query->ofPriority($priority);
        }

        if ($sprintSlug !== null) {
            $sprint = Sprint::forWorkspace($workspaceId)->where('slug', $sprintSlug)->first();
            if (! $sprint) {
                throw new \InvalidArgumentException("Sprint not found: {$sprintSlug}");
            }
            $query->forSprint($sprint->id);
        }

        if ($label !== null) {
            $query->withLabel($label);
        }

        return $query->get();
    }
}
