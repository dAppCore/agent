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
 * Get details of a specific phase within a plan.
 *
 * Resolves the phase by order number or name.
 *
 * Usage:
 *   $phase = GetPhase::run('deploy-v2', '1', 1);
 *   $phase = GetPhase::run('deploy-v2', 'Build', 1);
 */
class GetPhase
{
    use Action;

    /**
     * @throws \InvalidArgumentException
     */
    public function handle(string $planSlug, string|int $phase, int $workspaceId): AgentPhase
    {
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

        return $resolved;
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
