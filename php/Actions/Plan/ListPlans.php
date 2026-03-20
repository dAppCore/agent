<?php

/*
 * Core PHP Framework
 *
 * Licensed under the European Union Public Licence (EUPL) v1.2.
 * See LICENSE file for details.
 */

declare(strict_types=1);

namespace Core\Mod\Agentic\Actions\Plan;

use Core\Actions\Action;
use Core\Mod\Agentic\Models\AgentPlan;
use Illuminate\Support\Collection;

/**
 * List work plans for a workspace with optional filtering.
 *
 * Returns plans ordered by most recently updated, with progress data.
 *
 * Usage:
 *   $plans = ListPlans::run(1);
 *   $plans = ListPlans::run(1, 'active');
 */
class ListPlans
{
    use Action;

    /**
     * @return Collection<int, AgentPlan>
     */
    public function handle(int $workspaceId, ?string $status = null, bool $includeArchived = false): Collection
    {
        if ($status !== null) {
            $valid = ['draft', 'active', 'paused', 'completed', 'archived'];
            if (! in_array($status, $valid, true)) {
                throw new \InvalidArgumentException(
                    sprintf('status must be one of: %s', implode(', ', $valid))
                );
            }
        }

        $query = AgentPlan::with('agentPhases')
            ->forWorkspace($workspaceId)
            ->orderBy('updated_at', 'desc');

        if (! $includeArchived && $status !== 'archived') {
            $query->notArchived();
        }

        if ($status !== null) {
            $query->where('status', $status);
        }

        return $query->get();
    }
}
