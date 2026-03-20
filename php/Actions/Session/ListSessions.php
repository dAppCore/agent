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
use Illuminate\Support\Collection;

/**
 * List sessions for a workspace, with optional filtering.
 *
 * Usage:
 *   $sessions = ListSessions::run(1);
 *   $sessions = ListSessions::run(1, 'active', 'deploy-v2', 20);
 */
class ListSessions
{
    use Action;

    public function __construct(
        private AgentSessionService $sessionService,
    ) {}

    /**
     * @return Collection<int, AgentSession>
     */
    public function handle(int $workspaceId, ?string $status = null, ?string $planSlug = null, ?int $limit = null): Collection
    {
        if ($status !== null) {
            $valid = ['active', 'paused', 'completed', 'failed'];
            if (! in_array($status, $valid, true)) {
                throw new \InvalidArgumentException(
                    sprintf('status must be one of: %s', implode(', ', $valid))
                );
            }
        }

        // Active sessions use the optimised service method
        if ($status === 'active' || $status === null) {
            return $this->sessionService->getActiveSessions($workspaceId);
        }

        $query = AgentSession::query()
            ->where('workspace_id', $workspaceId)
            ->where('status', $status)
            ->orderBy('last_active_at', 'desc');

        if ($planSlug !== null) {
            $query->whereHas('plan', fn ($q) => $q->where('slug', $planSlug));
        }

        if ($limit !== null && $limit > 0) {
            $query->limit(min($limit, 1000));
        }

        return $query->get();
    }
}
