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
 * End an agent session with a final status and optional summary.
 *
 * Usage:
 *   $session = EndSession::run('ses_abc123', 'completed', 'All phases done');
 */
class EndSession
{
    use Action;

    public function __construct(
        private AgentSessionService $sessionService,
    ) {}

    /**
     * @throws \InvalidArgumentException
     */
    public function handle(string $sessionId, string $status, ?string $summary = null): AgentSession
    {
        if ($sessionId === '') {
            throw new \InvalidArgumentException('session_id is required');
        }

        $valid = ['completed', 'handed_off', 'paused', 'failed'];
        if (! in_array($status, $valid, true)) {
            throw new \InvalidArgumentException(
                sprintf('status must be one of: %s', implode(', ', $valid))
            );
        }

        $session = $this->sessionService->end($sessionId, $status, $summary);

        if (! $session) {
            throw new \InvalidArgumentException("Session not found: {$sessionId}");
        }

        return $session;
    }
}
