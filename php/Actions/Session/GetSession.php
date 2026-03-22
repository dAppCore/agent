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

/**
 * Get detailed information about a specific session.
 *
 * Returns the session with plan context, scoped to workspace.
 *
 * Usage:
 *   $session = GetSession::run('ses_abc123', 1);
 */
class GetSession
{
    use Action;

    /**
     * @throws \InvalidArgumentException
     */
    public function handle(string $sessionId, int $workspaceId): AgentSession
    {
        if ($sessionId === '') {
            throw new \InvalidArgumentException('session_id is required');
        }

        $session = AgentSession::with('plan')
            ->where('session_id', $sessionId)
            ->where('workspace_id', $workspaceId)
            ->first();

        if (! $session) {
            throw new \InvalidArgumentException("Session not found: {$sessionId}");
        }

        return $session;
    }
}
