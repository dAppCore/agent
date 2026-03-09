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

/**
 * Update the status of a plan.
 *
 * Validates the transition and updates the plan status.
 * Scoped to workspace for tenant isolation.
 *
 * Usage:
 *   $plan = UpdatePlanStatus::run('deploy-v2', 'active', 1);
 */
class UpdatePlanStatus
{
    use Action;

    /**
     * @throws \InvalidArgumentException
     */
    public function handle(string $slug, string $status, int $workspaceId): AgentPlan
    {
        $valid = ['draft', 'active', 'paused', 'completed'];
        if (! in_array($status, $valid, true)) {
            throw new \InvalidArgumentException(
                sprintf('status must be one of: %s', implode(', ', $valid))
            );
        }

        $plan = AgentPlan::forWorkspace($workspaceId)
            ->where('slug', $slug)
            ->first();

        if (! $plan) {
            throw new \InvalidArgumentException("Plan not found: {$slug}");
        }

        $plan->update(['status' => $status]);

        return $plan->fresh();
    }
}
