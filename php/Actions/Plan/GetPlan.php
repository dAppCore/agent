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
 * Get detailed information about a specific plan.
 *
 * Returns the plan with all phases, progress, and context data.
 * Scoped to workspace for tenant isolation.
 *
 * Usage:
 *   $plan = GetPlan::run('deploy-v2', 1);
 */
class GetPlan
{
    use Action;

    /**
     * @throws \InvalidArgumentException
     */
    public function handle(string $slug, int $workspaceId): AgentPlan
    {
        if ($slug === '') {
            throw new \InvalidArgumentException('slug is required');
        }

        $plan = AgentPlan::with('agentPhases')
            ->forWorkspace($workspaceId)
            ->where('slug', $slug)
            ->first();

        if (! $plan) {
            throw new \InvalidArgumentException("Plan not found: {$slug}");
        }

        return $plan;
    }
}
