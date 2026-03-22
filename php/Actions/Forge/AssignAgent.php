<?php

/*
 * Core PHP Framework
 *
 * Licensed under the European Union Public Licence (EUPL) v1.2.
 * See LICENSE file for details.
 */

declare(strict_types=1);

namespace Core\Mod\Agentic\Actions\Forge;

use Core\Actions\Action;
use Core\Mod\Agentic\Actions\Session\StartSession;
use Core\Mod\Agentic\Models\AgentPlan;
use Core\Mod\Agentic\Models\AgentSession;

/**
 * Assign an agent to a plan and start a session.
 *
 * Activates the plan if it is still in draft status, then
 * delegates to StartSession to create the working session.
 *
 * Usage:
 *   $session = AssignAgent::run($plan, 'opus', $workspaceId);
 */
class AssignAgent
{
    use Action;

    public function handle(AgentPlan $plan, string $agentType, int $workspaceId): AgentSession
    {
        if ($plan->status !== AgentPlan::STATUS_ACTIVE) {
            $plan->activate();
        }

        return StartSession::run($agentType, $plan->slug, $workspaceId);
    }
}
