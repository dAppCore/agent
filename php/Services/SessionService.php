<?php

// SPDX-License-Identifier: EUPL-1.2

declare(strict_types=1);

namespace Core\Mod\Agentic\Services;

use Core\Mod\Agentic\Models\AgentPlan;
use Core\Mod\Agentic\Models\AgentSession;
use Core\Tenant\Models\Workspace;
use Illuminate\Support\Facades\Event;
use InvalidArgumentException;

class SessionService
{
    public const SSE_EVENT = 'agentic.session.sse';

    private const STATE_ALIASES = [
        'closed' => AgentSession::STATUS_COMPLETED,
        'resumed' => AgentSession::STATUS_ACTIVE,
    ];

    private const TRANSITIONS = [
        AgentSession::STATUS_ACTIVE => [
            AgentSession::STATUS_ACTIVE,
            AgentSession::STATUS_PAUSED,
            AgentSession::STATUS_COMPLETED,
            AgentSession::STATUS_FAILED,
            AgentSession::STATUS_HANDED_OFF,
        ],
        AgentSession::STATUS_PAUSED => [
            AgentSession::STATUS_ACTIVE,
            AgentSession::STATUS_PAUSED,
            AgentSession::STATUS_COMPLETED,
            AgentSession::STATUS_FAILED,
            AgentSession::STATUS_HANDED_OFF,
        ],
        AgentSession::STATUS_HANDED_OFF => [
            AgentSession::STATUS_ACTIVE,
            AgentSession::STATUS_HANDED_OFF,
        ],
        AgentSession::STATUS_COMPLETED => [
            AgentSession::STATUS_COMPLETED,
        ],
        AgentSession::STATUS_FAILED => [
            AgentSession::STATUS_FAILED,
        ],
    ];

    public function create(Workspace|int $workspace, array $request): AgentSession
    {
        $workspaceId = $this->resolveWorkspaceId($workspace);
        $agentType = isset($request['agent_type']) ? trim((string) $request['agent_type']) : null;
        $plan = $this->resolvePlan($workspaceId, $request);
        $workspaceModel = $workspace instanceof Workspace ? $workspace : Workspace::query()->find($workspaceId);

        $session = AgentSession::start(
            $plan,
            $agentType !== '' ? $agentType : null,
            $workspaceModel instanceof Workspace ? $workspaceModel : null,
        );

        $attributes = [
            'workspace_id' => $workspaceId,
        ];

        if (isset($request['context_summary']) && is_array($request['context_summary'])) {
            $attributes['context_summary'] = $request['context_summary'];
        } elseif (isset($request['context']) && is_array($request['context'])) {
            $attributes['context_summary'] = $request['context'];
        }

        if (isset($request['work_log']) && is_array($request['work_log'])) {
            $attributes['work_log'] = array_values($request['work_log']);
        }

        if (isset($request['artifacts']) && is_array($request['artifacts'])) {
            $attributes['artifacts'] = array_values($request['artifacts']);
        }

        if (isset($request['handoff_notes']) && is_array($request['handoff_notes'])) {
            $attributes['handoff_notes'] = $request['handoff_notes'];
        }

        $session->update($attributes);
        $fresh = $session->fresh() ?? $session;

        $this->emit($fresh, [
            'event' => 'session.created',
            'data' => [
                'agent_type' => $fresh->agent_type,
                'status' => $fresh->status,
            ],
        ]);

        return $fresh;
    }

    public function updateState(AgentSession|string $session, string $newState): AgentSession
    {
        $record = $this->resolveSession($session);
        $targetState = $this->normaliseState($newState);
        $currentState = $record->status;
        $allowed = self::TRANSITIONS[$currentState] ?? [];

        if (! in_array($targetState, $allowed, true)) {
            throw new InvalidArgumentException(
                sprintf('Invalid session transition [%s -> %s]', $currentState, $targetState)
            );
        }

        if ($targetState === $currentState) {
            return $record;
        }

        match ($targetState) {
            AgentSession::STATUS_ACTIVE => $record->resume(),
            AgentSession::STATUS_PAUSED => $record->pause(),
            AgentSession::STATUS_COMPLETED,
            AgentSession::STATUS_FAILED,
            AgentSession::STATUS_HANDED_OFF => $record->end($targetState, $this->defaultSummary($targetState)),
            default => throw new InvalidArgumentException(sprintf('Unsupported session state [%s]', $targetState)),
        };

        $fresh = $record->fresh() ?? $record;

        $this->emit($fresh, [
            'event' => 'session.state.updated',
            'data' => [
                'from' => $currentState,
                'to' => $fresh->status,
            ],
        ]);

        return $fresh;
    }

    public function emit(AgentSession|string $session, array|string $event): string
    {
        $record = $this->resolveSession($session);
        $payload = $this->normaliseEventPayload($record, $event);
        Event::dispatch(self::SSE_EVENT, [$payload]);

        return $payload['frame'];
    }

    private function resolveSession(AgentSession|string $session): AgentSession
    {
        if ($session instanceof AgentSession) {
            return $session->fresh() ?? $session;
        }

        $query = AgentSession::query()->where('session_id', $session);

        if (ctype_digit($session)) {
            $query->orWhere('id', (int) $session);
        }

        $resolved = $query->first();

        if (! $resolved instanceof AgentSession) {
            throw new InvalidArgumentException('Session not found');
        }

        return $resolved;
    }

    private function resolvePlan(int $workspaceId, array $request): ?AgentPlan
    {
        if (isset($request['plan_id'])) {
            $plan = AgentPlan::query()
                ->where('workspace_id', $workspaceId)
                ->find((int) $request['plan_id']);

            if (! $plan instanceof AgentPlan) {
                throw new InvalidArgumentException('Plan not found');
            }

            return $plan;
        }

        if (isset($request['plan_slug'])) {
            $plan = AgentPlan::query()
                ->where('workspace_id', $workspaceId)
                ->where('slug', (string) $request['plan_slug'])
                ->first();

            if (! $plan instanceof AgentPlan) {
                throw new InvalidArgumentException('Plan not found');
            }

            return $plan;
        }

        return null;
    }

    private function normaliseState(string $state): string
    {
        $normalised = strtolower(trim($state));

        if ($normalised === '') {
            throw new InvalidArgumentException('new_state is required');
        }

        return self::STATE_ALIASES[$normalised] ?? $normalised;
    }

    private function defaultSummary(string $state): string
    {
        return match ($state) {
            AgentSession::STATUS_COMPLETED => 'Session completed',
            AgentSession::STATUS_FAILED => 'Session failed',
            AgentSession::STATUS_HANDED_OFF => 'Session handed off',
            default => sprintf('Session moved to %s', $state),
        };
    }

    private function normaliseEventPayload(AgentSession $session, array|string $event): array
    {
        $name = is_array($event)
            ? trim((string) ($event['event'] ?? $event['type'] ?? 'message'))
            : trim($event);

        if ($name === '') {
            $name = 'message';
        }

        $data = is_array($event) && isset($event['data']) && is_array($event['data'])
            ? $event['data']
            : [];

        $payload = [
            'id' => sprintf('%s:%s:%s', $session->session_id, $name, now()->getTimestamp()),
            'event' => $name,
            'stream' => 'sessions.'.$session->session_id,
            'session_id' => $session->session_id,
            'workspace_id' => $session->workspace_id,
            'data' => $data + [
                'session_id' => $session->session_id,
                'status' => $session->status,
                'agent_type' => $session->agent_type,
                'emitted_at' => now()->toIso8601String(),
            ],
        ];

        $payload['frame'] = $this->formatFrame(
            $payload['id'],
            $payload['event'],
            $payload['data'],
        );

        return $payload;
    }

    private function formatFrame(string $id, string $event, array $data): string
    {
        return sprintf(
            "id: %s\nevent: %s\ndata: %s\n\n",
            $id,
            $event,
            json_encode($data, JSON_THROW_ON_ERROR),
        );
    }

    private function resolveWorkspaceId(Workspace|int $workspace): int
    {
        $workspaceId = $workspace instanceof Workspace ? (int) $workspace->id : (int) $workspace;

        if ($workspaceId <= 0) {
            throw new InvalidArgumentException('workspace_id is required');
        }

        return $workspaceId;
    }
}
