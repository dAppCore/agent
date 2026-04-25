<?php

// SPDX-License-Identifier: EUPL-1.2

declare(strict_types=1);

namespace Mod\Mcp\Services;

use Core\Mod\Agentic\Models\AgentPlan;
use Core\Mod\Agentic\Models\AgentSession;
use Illuminate\Support\Collection;
use Illuminate\Support\Facades\Cache;

final class AgentSessionService
{
    protected const CACHE_PREFIX = 'mcp_session:';

    public function start(
        string $agentType,
        ?AgentPlan $plan = null,
        ?int $workspaceId = null,
        array $initialContext = []
    ): AgentSession {
        $session = AgentSession::start($plan, $agentType);

        if ($workspaceId !== null) {
            $session->update(['workspace_id' => $workspaceId]);
        }

        if ($initialContext !== []) {
            $session->updateContextSummary($initialContext);
        }

        $this->cacheActiveSession($session);

        return $session;
    }

    public function get(string $sessionId): ?AgentSession
    {
        return AgentSession::query()->where('session_id', $sessionId)->first();
    }

    public function resume(string $sessionId): ?AgentSession
    {
        $session = $this->get($sessionId);

        if (! $session instanceof AgentSession) {
            return null;
        }

        if ($session->status === AgentSession::STATUS_PAUSED) {
            $session->resume();
        }

        $session->touchActivity();
        $this->cacheActiveSession($session);

        return $session;
    }

    public function getActiveSessions(?int $workspaceId = null): Collection
    {
        $query = AgentSession::query()->active();

        if ($workspaceId !== null) {
            $query->where('workspace_id', $workspaceId);
        }

        return $query->orderByDesc('last_active_at')->get();
    }

    public function getSessionsForPlan(AgentPlan $plan): Collection
    {
        return AgentSession::query()
            ->forPlan($plan)
            ->orderByDesc('created_at')
            ->get();
    }

    public function getLatestSessionForPlan(AgentPlan $plan): ?AgentSession
    {
        return AgentSession::query()
            ->forPlan($plan)
            ->orderByDesc('created_at')
            ->first();
    }

    public function end(string $sessionId, string $status = AgentSession::STATUS_COMPLETED, ?string $summary = null): ?AgentSession
    {
        $session = $this->get($sessionId);

        if (! $session instanceof AgentSession) {
            return null;
        }

        $session->end($status, $summary);
        $this->clearCachedSession($session);

        return $session;
    }

    public function pause(string $sessionId): ?AgentSession
    {
        $session = $this->get($sessionId);

        if (! $session instanceof AgentSession) {
            return null;
        }

        $session->pause();

        return $session;
    }

    public function prepareHandoff(string $sessionId, string $summary, array $nextSteps = [], array $blockers = [], array $contextForNext = []): ?AgentSession
    {
        $session = $this->get($sessionId);

        if (! $session instanceof AgentSession) {
            return null;
        }

        $session->prepareHandoff($summary, $nextSteps, $blockers, $contextForNext);

        return $session;
    }

    public function getHandoffContext(string $sessionId): ?array
    {
        return $this->get($sessionId)?->getHandoffContext();
    }

    public function continueFrom(string $previousSessionId, string $newAgentType): ?AgentSession
    {
        $previousSession = $this->get($previousSessionId);

        if (! $previousSession instanceof AgentSession) {
            return null;
        }

        $handoffContext = $previousSession->getHandoffContext();
        $handoffNotes = (array) ($handoffContext['handoff_notes'] ?? []);
        $contextForNext = (array) ($handoffNotes['context_for_next'] ?? []);

        $newSession = $this->start(
            $newAgentType,
            $previousSession->plan,
            $previousSession->workspace_id,
            [
                'continued_from' => $previousSession->session_id,
                'previous_agent' => $previousSession->agent_type,
                'handoff_notes' => $handoffNotes,
                'inherited_context' => $contextForNext !== [] ? $contextForNext : ($handoffContext['context_summary'] ?? null),
            ],
        );

        $previousSession->end(AgentSession::STATUS_HANDED_OFF, 'Handed off to '.$newAgentType, $previousSession->handoff_notes);
        $this->clearCachedSession($previousSession);

        return $newSession;
    }

    public function setState(string $sessionId, string $key, mixed $value, ?int $ttl = null): void
    {
        Cache::put(
            self::CACHE_PREFIX.$sessionId.':'.$key,
            $value,
            $ttl ?? $this->cacheTtl(),
        );
    }

    public function getState(string $sessionId, string $key, mixed $default = null): mixed
    {
        return Cache::get(self::CACHE_PREFIX.$sessionId.':'.$key, $default);
    }

    public function exists(string $sessionId): bool
    {
        return AgentSession::query()->where('session_id', $sessionId)->exists();
    }

    public function isActive(string $sessionId): bool
    {
        return $this->get($sessionId)?->isActive() ?? false;
    }

    public function getSessionStats(?int $workspaceId = null, int $days = 7): array
    {
        $query = AgentSession::query()->where('created_at', '>=', now()->subDays($days));

        if ($workspaceId !== null) {
            $query->where('workspace_id', $workspaceId);
        }

        $sessions = $query->get();
        $byStatus = $sessions->groupBy('status')->map->count()->toArray();
        $byAgentType = $sessions->groupBy('agent_type')->map->count()->toArray();
        $avgDuration = round((float) $sessions
            ->where('status', AgentSession::STATUS_COMPLETED)
            ->avg(static fn (AgentSession $session): int => $session->getDuration() ?? 0), 1);

        return [
            'total' => $sessions->count(),
            'active' => $sessions->where('status', AgentSession::STATUS_ACTIVE)->count(),
            'by_status' => $byStatus,
            'by_agent_type' => $byAgentType,
            'avg_duration_minutes' => $avgDuration,
            'period_days' => $days,
        ];
    }

    public function cleanupStaleSessions(int $hoursInactive = 24): int
    {
        $cutoff = now()->subHours($hoursInactive);
        $sessions = AgentSession::query()
            ->active()
            ->where('last_active_at', '<', $cutoff)
            ->get();

        foreach ($sessions as $session) {
            $session->fail('Session timed out due to inactivity');
            $this->clearCachedSession($session);
        }

        return $sessions->count();
    }

    protected function cacheActiveSession(AgentSession $session): void
    {
        Cache::put(self::CACHE_PREFIX.'active:'.$session->session_id, [
            'session_id' => $session->session_id,
            'agent_type' => $session->agent_type,
            'plan_id' => $session->agent_plan_id,
            'workspace_id' => $session->workspace_id,
            'started_at' => $session->started_at?->toIso8601String(),
        ], $this->cacheTtl());
    }

    protected function clearCachedSession(AgentSession $session): void
    {
        Cache::forget(self::CACHE_PREFIX.'active:'.$session->session_id);
    }

    protected function cacheTtl(): int
    {
        return (int) config('mcp.session.cache_ttl', 86400);
    }
}
