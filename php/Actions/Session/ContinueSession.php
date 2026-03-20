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
use Core\Mod\Agentic\Models\AgentSession;
use Core\Mod\Agentic\Services\AgentSessionService;

/**
 * Continue from a previous session (multi-agent handoff).
 *
 * Creates a new session with context inherited from the previous one
 * and marks the previous session as handed off.
 *
 * Usage:
 *   $session = ContinueSession::run('ses_abc123', 'opus');
 */
class ContinueSession
{
    use Action;

    public function __construct(
        private AgentSessionService $sessionService,
    ) {}

    /**
     * @throws \InvalidArgumentException
     */
    public function handle(string $previousSessionId, string $agentType): AgentSession
    {
        if ($previousSessionId === '') {
            throw new \InvalidArgumentException('previous_session_id is required');
        }

        if ($agentType === '') {
            throw new \InvalidArgumentException('agent_type is required');
        }

        $session = $this->sessionService->continueFrom($previousSessionId, $agentType);

        if (! $session) {
            throw new \InvalidArgumentException("Previous session not found: {$previousSessionId}");
        }

        return $session;
    }
}
