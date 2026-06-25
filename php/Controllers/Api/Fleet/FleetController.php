<?php

// SPDX-License-Identifier: EUPL-1.2

declare(strict_types=1);

namespace Core\Mod\Agentic\Controllers\Api\Fleet;

use Core\Front\Controller;
use Core\Mod\Agentic\Models\DispatchJob;
use Core\Mod\Agentic\Services\DispatchService;
use Illuminate\Http\JsonResponse;
use Illuminate\Http\Request;
use Symfony\Component\HttpFoundation\StreamedResponse;

/**
 * /v1/fleet/dispatch + /v1/fleet/stream over the unified DispatchService — the
 * same agent_registrations + dispatch_jobs queue as the rest of the fleet API.
 */
class FleetController extends Controller
{
    public function __construct(
        private DispatchService $dispatch,
    ) {}

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

        $agentId = trim((string) ($validated['agent_id'] ?? ''));

        $job = $this->dispatch->enqueue((int) $request->attributes->get('workspace_id'), [
            'repo' => $validated['repo'],
            'branch' => $validated['branch'] ?? null,
            'task' => $validated['task'],
            'template' => $validated['template'] ?? null,
            'agent_type' => $validated['agent_model'] ?? null,
            'assigned_agent' => $agentId !== '' ? $agentId : null,
            'created_by' => $agentId !== '' ? $agentId : null,
            'report' => (isset($validated['report']) && is_array($validated['report'])) ? $validated['report'] : null,
        ]);

        return response()->json(['data' => $this->formatTask($job)], 201);
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
                $job = $this->dispatch->nextTask($workspaceId, $agentId, $capabilities);

                if ($job instanceof DispatchJob) {
                    $this->streamEvent('task.assigned', $this->formatTask($job));
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
