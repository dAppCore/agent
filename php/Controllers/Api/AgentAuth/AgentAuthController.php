<?php

// SPDX-License-Identifier: EUPL-1.2

declare(strict_types=1);

namespace Core\Mod\Agentic\Controllers\Api\AgentAuth;

use Core\Front\Controller;
use Core\Mod\Agentic\Models\AgentPlan;
use Core\Mod\Agentic\Models\AgentSession;
use Core\Mod\Agentic\Services\SessionService;
use Core\Tenant\Models\Workspace;
use Illuminate\Http\JsonResponse;
use Illuminate\Http\Request;

class AgentAuthController extends Controller
{
    public function register(Request $request): JsonResponse
    {
        $validated = $request->validate([
            'plan_id' => 'nullable|integer',
            'plan_slug' => 'nullable|string|max:255',
            'agent_type' => 'nullable|string|max:255',
            'context_summary' => 'nullable|array',
            'context' => 'nullable|array',
            'work_log' => 'nullable|array',
            'artifacts' => 'nullable|array',
            'handoff_notes' => 'nullable|array',
        ]);

        $session = $this->createSession(
            (int) $request->attributes->get('workspace_id'),
            $validated,
        );

        return response()->json(['data' => $this->formatSession($session)], 201);
    }

    /**
     * @param  array<string, mixed>  $payload
     */
    private function createSession(int $workspaceId, array $payload): AgentSession
    {
        $service = $this->resolveSessionService();

        if ($service !== null && method_exists($service, 'create')) {
            $session = $service->create($workspaceId, $payload);

            if ($session instanceof AgentSession) {
                return $session;
            }
        }

        $workspace = Workspace::query()->find($workspaceId);
        $agentType = trim((string) ($payload['agent_type'] ?? ''));
        $session = AgentSession::start(
            $this->resolvePlan($workspaceId, $payload),
            $agentType !== '' ? $agentType : null,
            $workspace instanceof Workspace ? $workspace : null,
        );

        $attributes = [];

        if (isset($payload['context_summary']) && is_array($payload['context_summary'])) {
            $attributes['context_summary'] = $payload['context_summary'];
        } elseif (isset($payload['context']) && is_array($payload['context'])) {
            $attributes['context_summary'] = $payload['context'];
        }

        if (isset($payload['work_log']) && is_array($payload['work_log'])) {
            $attributes['work_log'] = array_values($payload['work_log']);
        }

        if (isset($payload['artifacts']) && is_array($payload['artifacts'])) {
            $attributes['artifacts'] = array_values($payload['artifacts']);
        }

        if (isset($payload['handoff_notes']) && is_array($payload['handoff_notes'])) {
            $attributes['handoff_notes'] = $payload['handoff_notes'];
        }

        if ($attributes !== []) {
            $session->update($attributes);
        }

        return $session->fresh() ?? $session;
    }

    /**
     * @param  array<string, mixed>  $payload
     */
    private function resolvePlan(int $workspaceId, array $payload): ?AgentPlan
    {
        if (isset($payload['plan_id'])) {
            $plan = AgentPlan::query()
                ->where('workspace_id', $workspaceId)
                ->find((int) $payload['plan_id']);

            if (! $plan instanceof AgentPlan) {
                throw new \InvalidArgumentException('Plan not found');
            }

            return $plan;
        }

        if (isset($payload['plan_slug'])) {
            $plan = AgentPlan::query()
                ->where('workspace_id', $workspaceId)
                ->where('slug', (string) $payload['plan_slug'])
                ->first();

            if (! $plan instanceof AgentPlan) {
                throw new \InvalidArgumentException('Plan not found');
            }

            return $plan;
        }

        return null;
    }

    /**
     * @return array<string, mixed>
     */
    private function formatSession(AgentSession $session): array
    {
        return [
            'id' => $session->id,
            'session_id' => $session->session_id,
            'workspace_id' => $session->workspace_id,
            'agent_plan_id' => $session->agent_plan_id,
            'agent_type' => $session->agent_type,
            'status' => $session->status,
            'context_summary' => $session->context_summary ?? [],
            'work_log' => $session->work_log ?? [],
            'artifacts' => $session->artifacts ?? [],
            'handoff_notes' => $session->handoff_notes ?? [],
            'final_summary' => $session->final_summary,
            'started_at' => $session->started_at?->toIso8601String(),
            'last_active_at' => $session->last_active_at?->toIso8601String(),
            'ended_at' => $session->ended_at?->toIso8601String(),
        ];
    }

    private function resolveSessionService(): ?object
    {
        if (! class_exists(SessionService::class)) {
            return null;
        }

        $service = app(SessionService::class);

        return is_object($service) ? $service : null;
    }
}
