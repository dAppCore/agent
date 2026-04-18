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
 * Add a checkpoint note to a phase.
 *
 * Checkpoints record milestones, decisions, and progress notes
 * within a phase's metadata for later review.
 *
 * Usage:
 *   $phase = AddCheckpoint::run('deploy-v2', '1', 'Tests passing', 1);
 */
class AddCheckpoint
{
    use Action;

    /**
     * @throws \InvalidArgumentException
     */
    public function handle(string $planSlug, string|int $phase, string $note, int $workspaceId, array $context = []): AgentPhase
    {
        if ($note === '') {
            throw new \InvalidArgumentException('note is required');
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

        $resolved->addCheckpoint($note, $context);

        return $resolved->fresh();
    }

    private function resolvePhase(AgentPlan $plan, string|int $identifier): ?AgentPhase
    {
        if (is_numeric($identifier)) {
            return $plan->agentPhases()->where('order', (int) $identifier)->first();
        }

        return $plan->agentPhases()
            ->where('name', $identifier)
            ->first();
    }
}
