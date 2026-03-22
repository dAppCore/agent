<?php

/*
 * Core PHP Framework
 *
 * Licensed under the European Union Public Licence (EUPL) v1.2.
 * See LICENSE file for details.
 */

declare(strict_types=1);

namespace Core\Mod\Agentic\Actions\Phase;

use Core\Actions\Action;
use Core\Mod\Agentic\Models\AgentPhase;
use Core\Mod\Agentic\Models\AgentPlan;

/**
 * Update the status of a phase within a plan.
 *
 * Optionally adds a checkpoint note when the status changes.
 *
 * Usage:
 *   $phase = UpdatePhaseStatus::run('deploy-v2', '1', 'in_progress', 1);
 *   $phase = UpdatePhaseStatus::run('deploy-v2', 'Build', 'completed', 1, 'All tests pass');
 */
class UpdatePhaseStatus
{
    use Action;

    /**
     * @throws \InvalidArgumentException
     */
    public function handle(string $planSlug, string|int $phase, string $status, int $workspaceId, ?string $notes = null): AgentPhase
    {
        $valid = ['pending', 'in_progress', 'completed', 'blocked', 'skipped'];
        if (! in_array($status, $valid, true)) {
            throw new \InvalidArgumentException(
                sprintf('status must be one of: %s', implode(', ', $valid))
            );
        }

        $plan = AgentPlan::forWorkspace($workspaceId)
            ->where('slug', $planSlug)
            ->first();

        if (! $plan) {
            throw new \InvalidArgumentException("Plan not found: {$planSlug}");
        }

        $resolved = $this->resolvePhase($plan, $phase);

        if (! $resolved) {
            throw new \InvalidArgumentException("Phase not found: {$phase}");
        }

        if ($notes !== null && $notes !== '') {
            $resolved->addCheckpoint($notes, ['status_change' => $status]);
        }

        $resolved->update(['status' => $status]);

        return $resolved->fresh();
    }

    private function resolvePhase(AgentPlan $plan, string|int $identifier): ?AgentPhase
    {
        if (is_numeric($identifier)) {
            return $plan->agentPhases()->where('order', (int) $identifier)->first();
        }

        return $plan->agentPhases()
            ->where(function ($query) use ($identifier) {
                $query->where('name', $identifier)
                    ->orWhere('order', $identifier);
            })
            ->first();
    }
}
