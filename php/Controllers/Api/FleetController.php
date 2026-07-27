<?php

declare(strict_types=1);

namespace Core\Mod\Agentic\Controllers\Api;

use Core\Front\Controller;
use Core\Mod\Agentic\Models\AgentRegistration;
use Core\Mod\Agentic\Models\DispatchJob;
use Core\Mod\Agentic\Services\DispatchService;
use Illuminate\Http\JsonResponse;
use Illuminate\Http\Request;
use Symfony\Component\HttpFoundation\StreamedResponse;

/**
 * Fleet endpoints (/v1/fleet/*) over the unified DispatchService — agents land
 * in agent_registrations and tasks in dispatch_jobs, the same queue the console
 * and the MCP dispatch tool use. Response shapes are preserved for the Go
 * core-agent (task ids are now uuids rather than integers).
 */
class FleetController extends Controller
{
    public function __construct(
        private DispatchService $dispatch,
    ) {}

    public function register(Request $request): JsonResponse
    {
        $validated = $request->validate([
            'agent_id' => 'required|string|max:255',
            'platform' => 'required|string|max:64',
            'models' => 'nullable|array',
            'models.*' => 'string',
            'capabilities' => 'nullable|array',
        ]);

        $node = $this->dispatch->register((int) $request->attributes->get('workspace_id'), [
            'agent_id' => $validated['agent_id'],
            'platform' => $validated['platform'],
            'models' => $validated['models'] ?? [],
            'capabilities' => $validated['capabilities'] ?? [],
        ]);

        return response()->json(['data' => $this->formatNode($node)], 201);
    }

    public function heartbeat(Request $request): JsonResponse
    {
        $validated = $request->validate([
            'agent_id' => 'required|string|max:255',
            'status' => 'required|string|max:32',
            'compute_budget' => 'nullable|array',
        ]);

        $node = $this->dispatch->heartbeat(
            (int) $request->attributes->get('workspace_id'),
            $validated['agent_id'],
            $validated['status'],
            $validated['compute_budget'] ?? [],
        );

        if (! $node instanceof AgentRegistration) {
            return response()->json(['error' => 'not_registered', 'message' => 'Agent is not registered.'], 404);
        }

        return response()->json(['data' => $this->formatNode($node)]);
    }

    public function deregister(Request $request): JsonResponse
    {
        $validated = $request->validate([
            'agent_id' => 'required|string|max:255',
        ]);

        $this->dispatch->deregister((int) $request->attributes->get('workspace_id'), $validated['agent_id']);

        return response()->json(['data' => ['agent_id' => $validated['agent_id'], 'deregistered' => true]]);
    }

    public function index(Request $request): JsonResponse
    {
        $validated = $request->validate([
            'status' => 'nullable|string|max:32',
            'platform' => 'nullable|string|max:64',
        ]);

        $nodes = $this->dispatch->listAgents(
            (int) $request->attributes->get('workspace_id'),
            $validated['status'] ?? null,
            $validated['platform'] ?? null,
        );

        return response()->json([
            'data' => $nodes->map(fn (AgentRegistration $node) => $this->formatNode($node))->values()->all(),
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

        $job = $this->dispatch->enqueue((int) $request->attributes->get('workspace_id'), [
            'repo' => $validated['repo'],
            'branch' => $validated['branch'] ?? null,
            'task' => $validated['task'],
            'template' => $validated['template'] ?? null,
            'agent_type' => $validated['agent_model'] ?? null,
            'assigned_agent' => $validated['agent_id'],
            'created_by' => $validated['agent_id'],
        ]);

        return response()->json(['data' => $this->formatTask($job)], 201);
    }

    public function completeTask(Request $request): JsonResponse
    {
        $validated = $request->validate([
            'agent_id' => 'required|string|max:255',
            'task_id' => 'required|string|max:64',
            'status' => 'nullable|string|in:completed,failed',
            'result' => 'nullable|array',
            'findings' => 'nullable|array',
            'changes' => 'nullable|array',
            'report' => 'nullable|array',
        ]);

        $job = $this->dispatch->complete(
            (int) $request->attributes->get('workspace_id'),
            $validated['agent_id'],
            $validated['task_id'],
            [
                'status' => $validated['status'] ?? DispatchJob::STATUS_COMPLETED,
                'result' => $validated['result'] ?? [],
                'findings' => $validated['findings'] ?? [],
                'changes' => $validated['changes'] ?? [],
                'report' => $validated['report'] ?? [],
            ],
        );

        if (! $job instanceof DispatchJob) {
            return response()->json(['error' => 'not_found', 'message' => 'No matching job assigned to this agent.'], 404);
        }

        return response()->json(['data' => $this->formatTask($job)]);
    }

    public function nextTask(Request $request): JsonResponse
    {
        $validated = $request->validate([
            'agent_id' => 'required|string|max:255',
            'capabilities' => 'nullable|array',
        ]);

        $job = $this->dispatch->nextTask(
            (int) $request->attributes->get('workspace_id'),
            $validated['agent_id'],
            $validated['capabilities'] ?? [],
        );

        return response()->json(['data' => $job instanceof DispatchJob ? $this->formatTask($job) : null]);
    }

    public function events(Request $request): StreamedResponse
    {
        $validated = $request->validate([
            'agent_id' => 'required|string|max:255',
            'limit' => 'nullable|integer|min:1',
            'poll_interval_ms' => 'nullable|integer|min:100|max:5000',
        ]);

        $workspaceId = (int) $request->attributes->get('workspace_id');
        $agentId = $validated['agent_id'];
        $limit = $validated['limit'] ?? 0;
        $pollIntervalMs = $validated['poll_interval_ms'] ?? 1000;

        return response()->stream(function () use ($workspaceId, $agentId, $limit, $pollIntervalMs): void {
            $emitted = 0;

            ignore_user_abort(true);
            set_time_limit(0);

            $this->streamFleetEvent('ready', ['agent_id' => $agentId]);

            while (! connection_aborted()) {
                $job = $this->dispatch->nextTask($workspaceId, $agentId, []);
                if ($job instanceof DispatchJob) {
                    $this->streamFleetEvent('task.assigned', $this->formatTask($job));
                    $emitted++;

                    if ($limit > 0 && $emitted >= $limit) {
                        break;
                    }

                    continue;
                }

                usleep($pollIntervalMs * 1000);
            }
        }, 200, [
            'Content-Type' => 'text/event-stream',
            'Cache-Control' => 'no-cache',
            'Connection' => 'keep-alive',
            'X-Accel-Buffering' => 'no',
        ]);
    }

    /**
     * @param  array<string, mixed>  $data
     */
    private function streamFleetEvent(string $event, array $data): void
    {
        echo "event: {$event}\n";
        echo 'data: '.json_encode($data)."\n\n";

        @ob_flush();
        flush();
    }

    public function stats(Request $request): JsonResponse
    {
        $stats = $this->dispatch->stats((int) $request->attributes->get('workspace_id'));

        return response()->json(['data' => $stats]);
    }

    /**
     * @return array<string, mixed>
     */
    private function formatNode(AgentRegistration $node): array
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
            'registered_at' => $node->connected_at?->toIso8601String(),
        ];
    }

    /**
     * @return array<string, mixed>
     */
    private function formatTask(DispatchJob $job): array
    {
        return [
            'id' => $job->id,
            'repo' => $job->repo,
            'branch' => $job->branch,
            'task' => $job->task,
            'template' => $job->template,
            'agent_model' => $job->agent_type,
            'status' => $job->status,
            'result' => $job->result ?? [],
            'findings' => $job->findings ?? [],
            'changes' => $job->changes ?? [],
            'report' => $job->report ?? [],
            'started_at' => $job->started_at?->toIso8601String(),
            'completed_at' => $job->completed_at?->toIso8601String(),
        ];
    }
}
