<?php

declare(strict_types=1);

namespace Core\Mod\Agentic\Actions\Sprint;

use Core\Actions\Action;
use Core\Mod\Agentic\Models\Sprint;
use Illuminate\Support\Collection;

/**
 * List sprints for a workspace with optional filtering.
 *
 * Usage:
 *   $sprints = ListSprints::run(1);
 *   $sprints = ListSprints::run(1, 'active');
 */
class ListSprints
{
    use Action;

    /**
     * @return Collection<int, Sprint>
     */
    public function handle(int $workspaceId, ?string $status = null, bool $includeCancelled = false): Collection
    {
        $validStatuses = [Sprint::STATUS_PLANNING, Sprint::STATUS_ACTIVE, Sprint::STATUS_COMPLETED, Sprint::STATUS_CANCELLED];
        if ($status !== null && ! in_array($status, $validStatuses, true)) {
            throw new \InvalidArgumentException(
                sprintf('status must be one of: %s', implode(', ', $validStatuses))
            );
        }

        $query = Sprint::with('issues')
            ->forWorkspace($workspaceId)
            ->orderBy('updated_at', 'desc');

        if (! $includeCancelled && $status !== Sprint::STATUS_CANCELLED) {
            $query->notCancelled();
        }

        if ($status !== null) {
            $query->where('status', $status);
        }

        return $query->get();
    }
}
