<?php

/*
 * Core PHP Framework
 *
 * Licensed under the European Union Public Licence (EUPL) v1.2.
 * See LICENSE file for details.
 */

declare(strict_types=1);

namespace Core\Mod\Agentic\Actions\Session;

use Core\Actions\Action;
use Core\Mod\Agentic\Models\AgentPlan;
use Core\Mod\Agentic\Models\AgentSession;
use Core\Mod\Agentic\Services\AgentSessionService;

/**
 * Start a new agent session, optionally linked to a plan.
 *
 * Creates an active session and caches it for fast lookup.
 * Workspace can be provided directly or inferred from the plan.
 *
 * Usage:
 *   $session = StartSession::run('opus', null, 1);
 *   $session = StartSession::run('sonnet', 'deploy-v2', 1, ['goal' => 'testing']);
 */
class StartSession
{
    use Action;

    public function __construct(
        private AgentSessionService $sessionService,
    ) {}

    /**
     * @throws \InvalidArgumentException
     */
    public function handle(string $agentType, ?string $planSlug, int $workspaceId, array $context = []): AgentSession
    {
        if ($agentType === '') {
            throw new \InvalidArgumentException('agent_type is required');
        }

        $plan = null;
        if ($planSlug !== null && $planSlug !== '') {
            $plan = AgentPlan::where('slug', $planSlug)->first();
            if (! $plan) {
                throw new \InvalidArgumentException("Plan not found: {$planSlug}");
            }
        }

        return $this->sessionService->start($agentType, $plan, $workspaceId, $context);
    }
}
