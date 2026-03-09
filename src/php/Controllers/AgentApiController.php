<?php

declare(strict_types=1);

namespace Core\Mod\Agentic\Controllers;

use Core\Mod\Agentic\Models\AgentPlan;
use Core\Mod\Agentic\Models\AgentPhase;
use Core\Mod\Agentic\Models\AgentSession;
use Illuminate\Http\JsonResponse;
use Illuminate\Http\Request;
use Illuminate\Routing\Controller;

/**
 * Agent API Controller.
 *
 * REST endpoints consumed by the go-agentic Client (dispatch watch).
 * All routes are protected by AgentApiAuth middleware with Bearer token.
 *
 * Prefix: /api/v1
 */
class AgentApiController extends Controller
{
    // -------------------------------------------------------------------------
    // Health
    // -------------------------------------------------------------------------

    public function health(): JsonResponse
    {
        return response()->json([
            'status' => 'ok',
            'service' => 'core-agentic',
            'timestamp' => now()->toIso8601String(),
        ]);
    }

    // -------------------------------------------------------------------------
    // Plans
    // -------------------------------------------------------------------------

    /**
     * GET /v1/plans
     *
     * List plans with optional status filter.
     * Query params: status, include_archived
     */
    public function listPlans(Request $request): JsonResponse
    {
        $workspaceId = $request->attributes->get('workspace_id');

        $query = AgentPlan::where('workspace_id', $workspaceId);

        if ($status = $request->query('status')) {
            $query->where('status', $status);
        }

        if (! $request->boolean('include_archived')) {
            $query->notArchived();
        }

        $plans = $query->orderByStatus()->latest()->get();

        return response()->json([
            'plans' => $plans->map(fn (AgentPlan $p) => $this->formatPlan($p)),
            'total' => $plans->count(),
        ]);
    }

    /**
     * GET /v1/plans/{slug}
     *
     * Get plan detail with phases.
     */
    public function getPlan(Request $request, string $slug): JsonResponse
    {
        $workspaceId = $request->attributes->get('workspace_id');

        $plan = AgentPlan::where('workspace_id', $workspaceId)
            ->where('slug', $slug)
            ->first();

        if (! $plan) {
            return response()->json(['error' => 'not_found', 'message' => 'Plan not found'], 404);
        }

        return response()->json($this->formatPlanDetail($plan));
    }

    /**
     * POST /v1/plans
     *
     * Create a new plan with optional phases.
     */
    public function createPlan(Request $request): JsonResponse
    {
        $workspaceId = $request->attributes->get('workspace_id');

        $validated = $request->validate([
            'title' => 'required|string|max:255',
            'slug' => 'nullable|string|max:255',
            'description' => 'nullable|string',
            'context' => 'nullable|array',
            'phases' => 'nullable|array',
            'phases.*.name' => 'required|string',
            'phases.*.description' => 'nullable|string',
            'phases.*.tasks' => 'nullable|array',
        ]);

        $slug = $validated['slug'] ?? AgentPlan::generateSlug($validated['title']);

        $plan = AgentPlan::create([
            'workspace_id' => $workspaceId,
            'slug' => $slug,
            'title' => $validated['title'],
            'description' => $validated['description'] ?? null,
            'context' => $validated['context'] ?? null,
            'status' => AgentPlan::STATUS_DRAFT,
        ]);

        // Create phases if provided
        $phaseCount = 0;
        if (! empty($validated['phases'])) {
            foreach ($validated['phases'] as $order => $phaseData) {
                $tasks = [];
                foreach ($phaseData['tasks'] ?? [] as $taskName) {
                    $tasks[] = ['name' => $taskName, 'status' => 'pending'];
                }

                AgentPhase::create([
                    'agent_plan_id' => $plan->id,
                    'order' => $order,
                    'name' => $phaseData['name'],
                    'description' => $phaseData['description'] ?? null,
                    'tasks' => $tasks ?: null,
                    'status' => AgentPhase::STATUS_PENDING,
                ]);
                $phaseCount++;
            }
        }

        return response()->json([
            'slug' => $plan->slug,
            'title' => $plan->title,
            'status' => $plan->status,
            'phases' => $phaseCount,
        ], 201);
    }

    /**
     * PATCH /v1/plans/{slug}
     *
     * Update plan status.
     */
    public function updatePlan(Request $request, string $slug): JsonResponse
    {
        $workspaceId = $request->attributes->get('workspace_id');

        $plan = AgentPlan::where('workspace_id', $workspaceId)
            ->where('slug', $slug)
            ->first();

        if (! $plan) {
            return response()->json(['error' => 'not_found', 'message' => 'Plan not found'], 404);
        }

        $validated = $request->validate([
            'status' => 'required|string|in:draft,active,completed,archived',
        ]);

        match ($validated['status']) {
            'active' => $plan->activate(),
            'completed' => $plan->complete(),
            'archived' => $plan->archive(),
            default => $plan->update(['status' => $validated['status']]),
        };

        return response()->json([
            'slug' => $plan->slug,
            'status' => $plan->fresh()->status,
        ]);
    }

    /**
     * DELETE /v1/plans/{slug}
     *
     * Archive a plan with optional reason.
     */
    public function archivePlan(Request $request, string $slug): JsonResponse
    {
        $workspaceId = $request->attributes->get('workspace_id');

        $plan = AgentPlan::where('workspace_id', $workspaceId)
            ->where('slug', $slug)
            ->first();

        if (! $plan) {
            return response()->json(['error' => 'not_found', 'message' => 'Plan not found'], 404);
        }

        $reason = $request->input('reason');
        $plan->archive($reason);

        return response()->json([
            'slug' => $plan->slug,
            'status' => 'archived',
            'archived_at' => now()->toIso8601String(),
        ]);
    }

    // -------------------------------------------------------------------------
    // Phases
    // -------------------------------------------------------------------------

    /**
     * GET /v1/plans/{slug}/phases/{phase}
     *
     * Get a phase by order number.
     */
    public function getPhase(Request $request, string $slug, string $phase): JsonResponse
    {
        $plan = $this->findPlan($request, $slug);
        if (! $plan) {
            return response()->json(['error' => 'not_found', 'message' => 'Plan not found'], 404);
        }

        $agentPhase = $plan->agentPhases()->where('order', (int) $phase)->first();
        if (! $agentPhase) {
            return response()->json(['error' => 'not_found', 'message' => 'Phase not found'], 404);
        }

        return response()->json($this->formatPhase($agentPhase));
    }

    /**
     * PATCH /v1/plans/{slug}/phases/{phase}
     *
     * Update phase status and/or notes.
     */
    public function updatePhase(Request $request, string $slug, string $phase): JsonResponse
    {
        $plan = $this->findPlan($request, $slug);
        if (! $plan) {
            return response()->json(['error' => 'not_found', 'message' => 'Plan not found'], 404);
        }

        $agentPhase = $plan->agentPhases()->where('order', (int) $phase)->first();
        if (! $agentPhase) {
            return response()->json(['error' => 'not_found', 'message' => 'Phase not found'], 404);
        }

        $status = $request->input('status');
        $notes = $request->input('notes');

        if ($status) {
            match ($status) {
                'in_progress' => $agentPhase->start(),
                'completed' => $agentPhase->complete(),
                'blocked' => $agentPhase->block($notes),
                'skipped' => $agentPhase->skip($notes),
                'pending' => $agentPhase->reset(),
                default => null,
            };
        }

        if ($notes && ! in_array($status, ['blocked', 'skipped'])) {
            $agentPhase->addCheckpoint($notes);
        }

        return response()->json([
            'slug' => $slug,
            'phase' => (int) $phase,
            'status' => $agentPhase->fresh()->status,
        ]);
    }

    /**
     * POST /v1/plans/{slug}/phases/{phase}/checkpoint
     *
     * Add a checkpoint to a phase.
     */
    public function addCheckpoint(Request $request, string $slug, string $phase): JsonResponse
    {
        $plan = $this->findPlan($request, $slug);
        if (! $plan) {
            return response()->json(['error' => 'not_found', 'message' => 'Plan not found'], 404);
        }

        $agentPhase = $plan->agentPhases()->where('order', (int) $phase)->first();
        if (! $agentPhase) {
            return response()->json(['error' => 'not_found', 'message' => 'Phase not found'], 404);
        }

        $validated = $request->validate([
            'note' => 'required|string',
            'context' => 'nullable|array',
        ]);

        $agentPhase->addCheckpoint($validated['note'], $validated['context'] ?? []);

        return response()->json([
            'slug' => $slug,
            'phase' => (int) $phase,
            'checkpoints' => count($agentPhase->fresh()->getCheckpoints()),
        ]);
    }

    /**
     * PATCH /v1/plans/{slug}/phases/{phase}/tasks/{taskIdx}
     *
     * Update a task within a phase.
     */
    public function updateTask(Request $request, string $slug, string $phase, int $taskIdx): JsonResponse
    {
        $plan = $this->findPlan($request, $slug);
        if (! $plan) {
            return response()->json(['error' => 'not_found', 'message' => 'Plan not found'], 404);
        }

        $agentPhase = $plan->agentPhases()->where('order', (int) $phase)->first();
        if (! $agentPhase) {
            return response()->json(['error' => 'not_found', 'message' => 'Phase not found'], 404);
        }

        $tasks = $agentPhase->tasks ?? [];
        if (! isset($tasks[$taskIdx])) {
            return response()->json(['error' => 'not_found', 'message' => 'Task not found'], 404);
        }

        $status = $request->input('status');
        $notes = $request->input('notes');

        if (is_string($tasks[$taskIdx])) {
            $tasks[$taskIdx] = ['name' => $tasks[$taskIdx], 'status' => $status ?? 'pending'];
        } else {
            if ($status) {
                $tasks[$taskIdx]['status'] = $status;
            }
        }

        if ($notes) {
            $tasks[$taskIdx]['notes'] = $notes;
        }

        $agentPhase->update(['tasks' => $tasks]);

        return response()->json([
            'slug' => $slug,
            'phase' => (int) $phase,
            'task' => $taskIdx,
            'status' => $tasks[$taskIdx]['status'] ?? 'pending',
        ]);
    }

    /**
     * POST /v1/plans/{slug}/phases/{phase}/tasks/{taskIdx}/toggle
     *
     * Toggle a task between pending and completed.
     */
    public function toggleTask(Request $request, string $slug, string $phase, int $taskIdx): JsonResponse
    {
        $plan = $this->findPlan($request, $slug);
        if (! $plan) {
            return response()->json(['error' => 'not_found', 'message' => 'Plan not found'], 404);
        }

        $agentPhase = $plan->agentPhases()->where('order', (int) $phase)->first();
        if (! $agentPhase) {
            return response()->json(['error' => 'not_found', 'message' => 'Phase not found'], 404);
        }

        $tasks = $agentPhase->tasks ?? [];
        if (! isset($tasks[$taskIdx])) {
            return response()->json(['error' => 'not_found', 'message' => 'Task not found'], 404);
        }

        if (is_string($tasks[$taskIdx])) {
            $tasks[$taskIdx] = ['name' => $tasks[$taskIdx], 'status' => 'completed'];
        } else {
            $current = $tasks[$taskIdx]['status'] ?? 'pending';
            $tasks[$taskIdx]['status'] = $current === 'completed' ? 'pending' : 'completed';
        }

        $agentPhase->update(['tasks' => $tasks]);

        return response()->json([
            'slug' => $slug,
            'phase' => (int) $phase,
            'task' => $taskIdx,
            'status' => $tasks[$taskIdx]['status'] ?? 'pending',
        ]);
    }

    // -------------------------------------------------------------------------
    // Sessions
    // -------------------------------------------------------------------------

    /**
     * GET /v1/sessions
     *
     * List sessions with optional filters.
     * Query params: status, plan_slug, limit
     */
    public function listSessions(Request $request): JsonResponse
    {
        $workspaceId = $request->attributes->get('workspace_id');

        $query = AgentSession::where('workspace_id', $workspaceId);

        if ($status = $request->query('status')) {
            $query->where('status', $status);
        }

        if ($planSlug = $request->query('plan_slug')) {
            $plan = AgentPlan::where('workspace_id', $workspaceId)
                ->where('slug', $planSlug)
                ->first();
            if ($plan) {
                $query->where('agent_plan_id', $plan->id);
            } else {
                return response()->json(['sessions' => [], 'total' => 0]);
            }
        }

        $limit = (int) ($request->query('limit') ?: 50);
        $sessions = $query->latest('started_at')->limit($limit)->get();

        return response()->json([
            'sessions' => $sessions->map(fn (AgentSession $s) => $this->formatSession($s)),
            'total' => $sessions->count(),
        ]);
    }

    /**
     * GET /v1/sessions/{sessionId}
     *
     * Get session detail.
     */
    public function getSession(Request $request, string $sessionId): JsonResponse
    {
        $workspaceId = $request->attributes->get('workspace_id');

        $session = AgentSession::where('workspace_id', $workspaceId)
            ->where('session_id', $sessionId)
            ->first();

        if (! $session) {
            return response()->json(['error' => 'not_found', 'message' => 'Session not found'], 404);
        }

        return response()->json($this->formatSession($session));
    }

    /**
     * POST /v1/sessions
     *
     * Start a new session.
     */
    public function startSession(Request $request): JsonResponse
    {
        $workspaceId = $request->attributes->get('workspace_id');
        $apiKey = $request->attributes->get('agent_api_key');

        $validated = $request->validate([
            'agent_type' => 'required|string',
            'plan_slug' => 'nullable|string',
            'context' => 'nullable|array',
        ]);

        $plan = null;
        if (! empty($validated['plan_slug'])) {
            $plan = AgentPlan::where('workspace_id', $workspaceId)
                ->where('slug', $validated['plan_slug'])
                ->first();
        }

        $session = AgentSession::create([
            'workspace_id' => $workspaceId,
            'agent_api_key_id' => $apiKey?->id,
            'agent_plan_id' => $plan?->id,
            'session_id' => 'sess_' . \Ramsey\Uuid\Uuid::uuid4()->toString(),
            'agent_type' => $validated['agent_type'],
            'status' => AgentSession::STATUS_ACTIVE,
            'context_summary' => $validated['context'] ?? [],
            'work_log' => [],
            'artifacts' => [],
            'started_at' => now(),
            'last_active_at' => now(),
        ]);

        return response()->json([
            'session_id' => $session->session_id,
            'agent_type' => $session->agent_type,
            'plan' => $plan?->slug,
            'status' => $session->status,
        ], 201);
    }

    /**
     * POST /v1/sessions/{sessionId}/end
     *
     * End a session.
     */
    public function endSession(Request $request, string $sessionId): JsonResponse
    {
        $workspaceId = $request->attributes->get('workspace_id');

        $session = AgentSession::where('workspace_id', $workspaceId)
            ->where('session_id', $sessionId)
            ->first();

        if (! $session) {
            return response()->json(['error' => 'not_found', 'message' => 'Session not found'], 404);
        }

        $validated = $request->validate([
            'status' => 'required|string|in:completed,failed',
            'summary' => 'nullable|string',
        ]);

        $session->end($validated['status'], $validated['summary'] ?? null);

        return response()->json([
            'session_id' => $session->session_id,
            'status' => $session->fresh()->status,
            'duration' => $session->getDurationFormatted(),
        ]);
    }

    /**
     * POST /v1/sessions/{sessionId}/continue
     *
     * Continue from a previous session (multi-agent handoff).
     */
    public function continueSession(Request $request, string $sessionId): JsonResponse
    {
        $workspaceId = $request->attributes->get('workspace_id');

        $previousSession = AgentSession::where('workspace_id', $workspaceId)
            ->where('session_id', $sessionId)
            ->first();

        if (! $previousSession) {
            return response()->json(['error' => 'not_found', 'message' => 'Session not found'], 404);
        }

        $validated = $request->validate([
            'agent_type' => 'required|string',
        ]);

        $newSession = $previousSession->createReplaySession($validated['agent_type']);

        return response()->json([
            'session_id' => $newSession->session_id,
            'agent_type' => $newSession->agent_type,
            'plan' => $newSession->plan?->slug,
            'status' => $newSession->status,
            'continued_from' => $previousSession->session_id,
        ], 201);
    }

    // -------------------------------------------------------------------------
    // Formatters (match go-agentic JSON contract)
    // -------------------------------------------------------------------------

    private function formatPlan(AgentPlan $plan): array
    {
        $progress = $plan->getProgress();

        return [
            'slug' => $plan->slug,
            'title' => $plan->title,
            'description' => $plan->description,
            'status' => $plan->status,
            'current_phase' => $plan->current_phase !== null ? (int) $plan->current_phase : null,
            'progress' => $progress,
            'metadata' => $plan->metadata,
            'created_at' => $plan->created_at?->toIso8601String(),
            'updated_at' => $plan->updated_at?->toIso8601String(),
        ];
    }

    private function formatPlanDetail(AgentPlan $plan): array
    {
        $data = $this->formatPlan($plan);
        $data['phases'] = $plan->agentPhases->map(fn (AgentPhase $p) => $this->formatPhase($p))->all();

        return $data;
    }

    private function formatPhase(AgentPhase $phase): array
    {
        $taskProgress = $phase->getTaskProgress();

        return [
            'id' => $phase->id,
            'order' => $phase->order,
            'name' => $phase->name,
            'description' => $phase->description,
            'status' => $phase->status,
            'tasks' => $phase->tasks,
            'task_progress' => [
                'total' => $taskProgress['total'],
                'completed' => $taskProgress['completed'],
                'pending' => $taskProgress['remaining'],
                'percentage' => (int) $taskProgress['percentage'],
            ],
            'remaining_tasks' => $phase->getRemainingTasks(),
            'dependencies' => $phase->dependencies,
            'dependency_blockers' => $phase->checkDependencies(),
            'can_start' => $phase->canStart(),
            'checkpoints' => $phase->getCheckpoints(),
            'started_at' => $phase->started_at?->toIso8601String(),
            'completed_at' => $phase->completed_at?->toIso8601String(),
            'metadata' => $phase->metadata,
        ];
    }

    private function formatSession(AgentSession $session): array
    {
        return [
            'session_id' => $session->session_id,
            'agent_type' => $session->agent_type,
            'status' => $session->status,
            'plan_slug' => $session->plan?->slug,
            'plan' => $session->plan?->slug,
            'duration' => $session->getDurationFormatted(),
            'started_at' => $session->started_at?->toIso8601String(),
            'last_active_at' => $session->last_active_at?->toIso8601String(),
            'ended_at' => $session->ended_at?->toIso8601String(),
            'action_count' => count($session->work_log ?? []),
            'artifact_count' => count($session->artifacts ?? []),
            'context_summary' => $session->context_summary,
            'handoff_notes' => $session->handoff_notes ? ($session->handoff_notes['summary'] ?? '') : null,
        ];
    }

    // -------------------------------------------------------------------------
    // Helpers
    // -------------------------------------------------------------------------

    private function findPlan(Request $request, string $slug): ?AgentPlan
    {
        $workspaceId = $request->attributes->get('workspace_id');

        return AgentPlan::where('workspace_id', $workspaceId)
            ->where('slug', $slug)
            ->first();
    }
}
