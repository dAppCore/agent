<?php

declare(strict_types=1);

namespace Core\Mod\Agentic\Controllers\Api;

use Core\Front\Controller;
use Core\Mod\Agentic\Actions\Fleet\AssignTask;
use Core\Mod\Agentic\Actions\Fleet\CompleteTask;
use Core\Mod\Agentic\Actions\Fleet\DeregisterNode;
use Core\Mod\Agentic\Actions\Fleet\GetFleetStats;
use Core\Mod\Agentic\Actions\Fleet\GetNextTask;
use Core\Mod\Agentic\Actions\Fleet\ListNodes;
use Core\Mod\Agentic\Actions\Fleet\NodeHeartbeat;
use Core\Mod\Agentic\Actions\Fleet\RegisterNode;
use Core\Mod\Agentic\Models\FleetNode;
use Core\Mod\Agentic\Models\FleetTask;
use Illuminate\Http\JsonResponse;
use Illuminate\Http\Request;
use Symfony\Component\HttpFoundation\StreamedResponse;

class FleetController extends Controller
{
    public function register(Request $request): JsonResponse
    {
        $validated = $request->validate([
            'agent_id' => 'required|string|max:255',
            'platform' => 'required|string|max:64',
            'models' => 'nullable|array',
            'models.*' => 'string',
            'capabilities' => 'nullable|array',
        ]);

        $node = RegisterNode::run(
            (int) $request->attributes->get('workspace_id'),
            $validated['agent_id'],
            $validated['platform'],
            $validated['models'] ?? [],
            $validated['capabilities'] ?? [],
        );

        return response()->json(['data' => $this->formatNode($node)], 201);
    }

    public function heartbeat(Request $request): JsonResponse
    {
        $validated = $request->validate([
            'agent_id' => 'required|string|max:255',
            'status' => 'required|string|max:32',
            'compute_budget' => 'nullable|array',
        ]);

        $node = NodeHeartbeat::run(
            (int) $request->attributes->get('workspace_id'),
            $validated['agent_id'],
            $validated['status'],
            $validated['compute_budget'] ?? [],
        );

        return response()->json(['data' => $this->formatNode($node)]);
    }

    public function deregister(Request $request): JsonResponse
    {
        $validated = $request->validate([
            'agent_id' => 'required|string|max:255',
        ]);

        DeregisterNode::run((int) $request->attributes->get('workspace_id'), $validated['agent_id']);

        return response()->json(['data' => ['agent_id' => $validated['agent_id'], 'deregistered' => true]]);
    }

    public function index(Request $request): JsonResponse
    {
        $validated = $request->validate([
            'status' => 'nullable|string|max:32',
            'platform' => 'nullable|string|max:64',
        ]);

        $nodes = ListNodes::run(
            (int) $request->attributes->get('workspace_id'),
            $validated['status'] ?? null,
            $validated['platform'] ?? null,
        );

        return response()->json([
            'data' => $nodes->map(fn (FleetNode $node) => $this->formatNode($node))->values()->all(),
            'total' => $nodes->count(),
        ]);
    }

    public function assignTask(Request $request): JsonResponse
    {
        $validated = $request->validate([
            'agent_id' => 'required|string|max:255',
            'repo' => 'required|string|max:255',
            'branch' => 'nullable|string|max:255',
            'task' => 'required|string|max:10000',
            'template' => 'nullable|string|max:255',
            'agent_model' => 'nullable|string|max:255',
        ]);

        $fleetTask = AssignTask::run(
            (int) $request->attributes->get('workspace_id'),
            $validated['agent_id'],
            $validated['task'],
            $validated['repo'],
            $validated['template'] ?? null,
            $validated['branch'] ?? null,
            $validated['agent_model'] ?? null,
        );

        return response()->json(['data' => $this->formatTask($fleetTask)], 201);
    }

    public function completeTask(Request $request): JsonResponse
    {
        $validated = $request->validate([
            'agent_id' => 'required|string|max:255',
            'task_id' => 'required|integer',
            'result' => 'nullable|array',
            'findings' => 'nullable|array',
            'changes' => 'nullable|array',
            'report' => 'nullable|array',
        ]);

        $fleetTask = CompleteTask::run(
            (int) $request->attributes->get('workspace_id'),
            $validated['agent_id'],
            (int) $validated['task_id'],
            $validated['result'] ?? [],
            $validated['findings'] ?? [],
            $validated['changes'] ?? [],
            $validated['report'] ?? [],
        );

        return response()->json(['data' => $this->formatTask($fleetTask)]);
    }

    public function nextTask(Request $request): JsonResponse
    {
        $validated = $request->validate([
            'agent_id' => 'required|string|max:255',
            'capabilities' => 'nullable|array',
        ]);

        $fleetTask = GetNextTask::run(
            (int) $request->attributes->get('workspace_id'),
            $validated['agent_id'],
            $validated['capabilities'] ?? [],
        );

        return response()->json(['data' => $fleetTask ? $this->formatTask($fleetTask) : null]);
    }

    public function events(Request $request): StreamedResponse
    {
        $validated = $request->validate([
            'agent_id' => 'required|string|max:255',
        ]);

        $workspaceId = (int) $request->attributes->get('workspace_id');
        $agentId = $validated['agent_id'];

        return response()->stream(function () use ($workspaceId, $agentId): void {
            echo "event: ready\n";
            echo 'data: '.json_encode(['agent_id' => $agentId])."\n\n";

            $fleetTask = GetNextTask::run($workspaceId, $agentId, []);
            if ($fleetTask instanceof FleetTask) {
                echo "event: task.assigned\n";
                echo 'data: '.json_encode($this->formatTask($fleetTask))."\n\n";
            }

            @ob_flush();
            flush();
        }, 200, [
            'Content-Type' => 'text/event-stream',
            'Cache-Control' => 'no-cache',
            'Connection' => 'keep-alive',
            'X-Accel-Buffering' => 'no',
        ]);
    }

    public function stats(Request $request): JsonResponse
    {
        $stats = GetFleetStats::run((int) $request->attributes->get('workspace_id'));

        return response()->json(['data' => $stats]);
    }

    /**
     * @return array<string, mixed>
     */
    private function formatNode(FleetNode $node): array
    {
        return [
            'id' => $node->id,
            'agent_id' => $node->agent_id,
            'platform' => $node->platform,
            'models' => $node->models ?? [],
            'capabilities' => $node->capabilities ?? [],
            'status' => $node->status,
            'compute_budget' => $node->compute_budget ?? [],
            'current_task_id' => $node->current_task_id,
            'last_heartbeat_at' => $node->last_heartbeat_at?->toIso8601String(),
            'registered_at' => $node->registered_at?->toIso8601String(),
        ];
    }

    /**
     * @return array<string, mixed>
     */
    private function formatTask(FleetTask $fleetTask): array
    {
        return [
            'id' => $fleetTask->id,
            'repo' => $fleetTask->repo,
            'branch' => $fleetTask->branch,
            'task' => $fleetTask->task,
            'template' => $fleetTask->template,
            'agent_model' => $fleetTask->agent_model,
            'status' => $fleetTask->status,
            'result' => $fleetTask->result ?? [],
            'findings' => $fleetTask->findings ?? [],
            'changes' => $fleetTask->changes ?? [],
            'report' => $fleetTask->report ?? [],
            'started_at' => $fleetTask->started_at?->toIso8601String(),
            'completed_at' => $fleetTask->completed_at?->toIso8601String(),
        ];
    }
}
