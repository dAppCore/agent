<?php

// SPDX-License-Identifier: EUPL-1.2

declare(strict_types=1);

namespace Core\Mod\Agentic\Controllers\Api\Fleet;

use Core\Front\Controller;
use Core\Mod\Agentic\Actions\Fleet\AssignTask;
use Core\Mod\Agentic\Actions\Fleet\GetNextTask;
use Core\Mod\Agentic\Models\FleetTask;
use Core\Mod\Agentic\Services\FleetService;
use Illuminate\Http\JsonResponse;
use Illuminate\Http\Request;
use Symfony\Component\HttpFoundation\StreamedResponse;

class FleetController extends Controller
{
    public function dispatch(Request $request): JsonResponse
    {
        $validated = $request->validate([
            'agent_id' => 'nullable|string|max:255',
            'repo' => 'required|string|max:255',
            'branch' => 'nullable|string|max:255',
            'task' => 'required|string|max:10000',
            'template' => 'nullable|string|max:255',
            'agent_model' => 'nullable|string|max:255',
            'report' => 'nullable|array',
        ]);

        $fleetTask = $this->dispatchTask(
            (int) $request->attributes->get('workspace_id'),
            $validated,
        );

        return response()->json(['data' => $this->formatTask($fleetTask)], 201);
    }

    public function stream(Request $request): StreamedResponse
    {
        $validated = $request->validate([
            'agent_id' => 'required|string|max:255',
            'capabilities' => 'nullable|array',
            'capabilities.*' => 'string',
            'limit' => 'nullable|integer|min:1',
            'poll_interval_ms' => 'nullable|integer|min:100|max:5000',
        ]);

        $workspaceId = (int) $request->attributes->get('workspace_id');
        $agentId = $validated['agent_id'];
        $capabilities = $validated['capabilities'] ?? [];
        $limit = (int) ($validated['limit'] ?? 0);
        $pollIntervalMs = (int) ($validated['poll_interval_ms'] ?? 1000);

        return response()->stream(function () use ($workspaceId, $agentId, $capabilities, $limit, $pollIntervalMs): void {
            $emitted = 0;

            ignore_user_abort(true);
            set_time_limit(0);

            $this->streamEvent('ready', ['agent_id' => $agentId]);

            while (! connection_aborted()) {
                $fleetTask = GetNextTask::run($workspaceId, $agentId, $capabilities);

                if ($fleetTask instanceof FleetTask) {
                    $this->streamEvent('task.assigned', $this->formatTask($fleetTask));
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
     * @param  array<string, mixed>  $payload
     */
    private function dispatchTask(int $workspaceId, array $payload): FleetTask
    {
        $service = $this->resolveFleetService();

        if ($service !== null && method_exists($service, 'dispatch')) {
            $fleetTask = $service->dispatch($workspaceId, $payload);

            if ($fleetTask instanceof FleetTask) {
                return $fleetTask;
            }
        }

        $agentId = trim((string) ($payload['agent_id'] ?? ''));
        if ($agentId !== '') {
            return AssignTask::run(
                $workspaceId,
                $agentId,
                (string) $payload['task'],
                (string) $payload['repo'],
                isset($payload['template']) ? (string) $payload['template'] : null,
                isset($payload['branch']) ? (string) $payload['branch'] : null,
                isset($payload['agent_model']) ? (string) $payload['agent_model'] : null,
            );
        }

        $fleetTask = FleetTask::query()->create([
            'workspace_id' => $workspaceId,
            'fleet_node_id' => null,
            'repo' => (string) $payload['repo'],
            'branch' => isset($payload['branch']) ? (string) $payload['branch'] : null,
            'task' => (string) $payload['task'],
            'template' => isset($payload['template']) ? (string) $payload['template'] : null,
            'agent_model' => isset($payload['agent_model']) ? (string) $payload['agent_model'] : null,
            'status' => FleetTask::STATUS_QUEUED,
            'report' => isset($payload['report']) && is_array($payload['report']) ? $payload['report'] : null,
        ])->fresh();

        if (! $fleetTask instanceof FleetTask) {
            throw new \RuntimeException('Failed to create fleet task');
        }

        return $fleetTask;
    }

    /**
     * @param  array<string, mixed>  $data
     */
    private function streamEvent(string $event, array $data): void
    {
        echo "event: {$event}\n";
        echo 'data: '.json_encode($data)."\n\n";

        @ob_flush();
        flush();
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

    private function resolveFleetService(): ?object
    {
        if (! class_exists(FleetService::class)) {
            return null;
        }

        $service = app(FleetService::class);

        return is_object($service) ? $service : null;
    }
}
