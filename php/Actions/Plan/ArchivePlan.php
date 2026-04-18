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
 * Archive a completed or abandoned plan.
 *
 * Sets the plan status to archived with an optional reason.
 * Scoped to workspace for tenant isolation.
 *
 * Usage:
 *   $plan = ArchivePlan::run('deploy-v2', 1, 'Superseded by v3');
 */
class ArchivePlan
{
    use Action;

    /**
     * @throws \InvalidArgumentException
     */
    public function handle(string $slug, int $workspaceId, ?string $reason = null): AgentPlan
    {
        if ($slug === '') {
            throw new \InvalidArgumentException('slug is required');
        }

        $plan = AgentPlan::forWorkspace($workspaceId)
            ->where('slug', $slug)
            ->first();

        if (! $plan) {
            throw new \InvalidArgumentException("Plan not found: {$slug}");
        }

        $plan->archive($reason);

        return $plan->fresh();
    }
}
