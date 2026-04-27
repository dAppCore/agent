<?php

// SPDX-License-Identifier: EUPL-1.2

declare(strict_types=1);

namespace Core\Mod\Agentic\Services;

use Core\Mod\Agentic\Actions\Fleet\AssignTask;
use Core\Mod\Agentic\Actions\Fleet\GetFleetStats;
use Core\Mod\Agentic\Actions\Fleet\NodeHeartbeat;
use Core\Mod\Agentic\Actions\Fleet\RegisterNode;
use Core\Mod\Agentic\Data\FleetStats;
use Core\Mod\Agentic\Models\FleetNode;
use Core\Mod\Agentic\Models\FleetTask;
use Core\Tenant\Models\Workspace;
use InvalidArgumentException;

class FleetService
{
    public function register(array|FleetNode $node): FleetNode
    {
        $payload = $this->normaliseNodePayload($node);

        return RegisterNode::run(
            $payload['workspace_id'],
            $payload['agent_id'],
            $payload['platform'],
            $payload['models'],
            $payload['capabilities'],
        );
    }

    public function heartbeat(array|FleetNode $node): FleetNode
    {
        $payload = $this->normaliseNodePayload($node);

        return NodeHeartbeat::run(
            $payload['workspace_id'],
            $payload['agent_id'],
            $payload['status'],
            $payload['compute_budget'],
        );
    }

    public function dispatch(Workspace|int $workspace, array $task): FleetTask
    {
        $workspaceId = $this->resolveWorkspaceId($workspace);
        $repo = trim((string) ($task['repo'] ?? ''));
        $description = trim((string) ($task['task'] ?? ''));

        if ($repo === '' || $description === '') {
            throw new InvalidArgumentException('repo and task are required');
        }

        $agentId = trim((string) ($task['agent_id'] ?? ''));
        if ($agentId !== '') {
            return AssignTask::run(
                $workspaceId,
                $agentId,
                $description,
                $repo,
                isset($task['template']) ? (string) $task['template'] : null,
                isset($task['branch']) ? (string) $task['branch'] : null,
                isset($task['agent_model']) ? (string) $task['agent_model'] : null,
            );
        }

        return FleetTask::query()->create([
            'workspace_id' => $workspaceId,
            'fleet_node_id' => null,
            'repo' => $repo,
            'branch' => isset($task['branch']) ? (string) $task['branch'] : null,
            'task' => $description,
            'template' => isset($task['template']) ? (string) $task['template'] : null,
            'agent_model' => isset($task['agent_model']) ? (string) $task['agent_model'] : null,
            'status' => FleetTask::STATUS_QUEUED,
            'report' => isset($task['report']) && is_array($task['report']) ? $task['report'] : null,
        ])->fresh();
    }

    public function health(FleetNode|array|int|string $node): array
    {
        $fleetNode = $this->resolveNode($node);
        $lastHeartbeat = $fleetNode->last_heartbeat_at;
        $ageSeconds = $lastHeartbeat?->diffInSeconds(now());
        $pendingTasks = FleetTask::query()
            ->pendingForNode($fleetNode)
            ->count();

        return [
            'id' => $fleetNode->id,
            'workspace_id' => $fleetNode->workspace_id,
            'agent_id' => $fleetNode->agent_id,
            'status' => $fleetNode->status,
            'is_online' => in_array($fleetNode->status, [FleetNode::STATUS_ONLINE, FleetNode::STATUS_BUSY], true),
            'is_stale' => $ageSeconds === null || $ageSeconds > 300,
            'last_heartbeat_at' => $lastHeartbeat?->toIso8601String(),
            'last_heartbeat_age_seconds' => $ageSeconds,
            'current_task_id' => $fleetNode->current_task_id,
            'pending_tasks' => $pendingTasks,
            'compute_budget' => $fleetNode->compute_budget ?? [],
        ];
    }

    public function stats(Workspace|int|null $workspace = null): FleetStats
    {
        if ($workspace !== null) {
            return FleetStats::fromArray(
                GetFleetStats::run($this->resolveWorkspaceId($workspace))
            );
        }

        $nodes = FleetNode::query();
        $tasks = FleetTask::query();
        $taskSamples = (clone $tasks)
            ->whereNotNull('started_at')
            ->get();

        return FleetStats::fromArray([
            'nodes_online' => (clone $nodes)->online()->count(),
            'tasks_today' => (clone $tasks)->whereDate('created_at', today())->count(),
            'tasks_week' => (clone $tasks)->where('created_at', '>=', now()->subDays(7))->count(),
            'repos_touched' => (clone $tasks)->distinct('repo')->count('repo'),
            'findings_total' => (clone $tasks)->get()->sum(
                static fn (FleetTask $fleetTask): int => count($fleetTask->findings ?? [])
            ),
            'compute_hours' => (int) round(
                $taskSamples->sum(fn (FleetTask $fleetTask): int => $this->taskDurationSeconds($fleetTask)) / 3600,
            ),
        ]);
    }

    private function normaliseNodePayload(array|FleetNode $node): array
    {
        $payload = $node instanceof FleetNode ? $node->getAttributes() + [
            'models' => $node->models ?? [],
            'capabilities' => $node->capabilities ?? [],
            'compute_budget' => $node->compute_budget ?? [],
            'status' => $node->status,
        ] : $node;

        $workspaceId = $this->resolveWorkspaceId($payload['workspace'] ?? $payload['workspace_id'] ?? null);
        $agentId = trim((string) ($payload['agent_id'] ?? ''));

        if ($agentId === '') {
            throw new InvalidArgumentException('agent_id is required');
        }

        return [
            'workspace_id' => $workspaceId,
            'agent_id' => $agentId,
            'platform' => trim((string) ($payload['platform'] ?? 'unknown')) ?: 'unknown',
            'models' => array_values((array) ($payload['models'] ?? [])),
            'capabilities' => (array) ($payload['capabilities'] ?? []),
            'status' => trim((string) ($payload['status'] ?? FleetNode::STATUS_ONLINE)) ?: FleetNode::STATUS_ONLINE,
            'compute_budget' => (array) ($payload['compute_budget'] ?? []),
        ];
    }

    private function resolveNode(FleetNode|array|int|string $node): FleetNode
    {
        if ($node instanceof FleetNode) {
            return $node->fresh() ?? $node;
        }

        if (is_array($node)) {
            if (isset($node['id'])) {
                $resolved = FleetNode::query()->find((int) $node['id']);
                if ($resolved instanceof FleetNode) {
                    return $resolved;
                }
            }

            $workspaceId = $this->resolveWorkspaceId($node['workspace'] ?? $node['workspace_id'] ?? null);
            $agentId = trim((string) ($node['agent_id'] ?? ''));

            $resolved = FleetNode::query()
                ->where('workspace_id', $workspaceId)
                ->where('agent_id', $agentId)
                ->first();

            if ($resolved instanceof FleetNode) {
                return $resolved;
            }

            throw new InvalidArgumentException('Fleet node not found');
        }

        $resolved = is_int($node)
            ? FleetNode::query()->find($node)
            : FleetNode::query()->where('agent_id', (string) $node)->first();

        if (! $resolved instanceof FleetNode) {
            throw new InvalidArgumentException('Fleet node not found');
        }

        return $resolved;
    }

    private function resolveWorkspaceId(Workspace|int|null $workspace): int
    {
        $workspaceId = $workspace instanceof Workspace ? (int) $workspace->id : (int) $workspace;

        if ($workspaceId <= 0) {
            throw new InvalidArgumentException('workspace_id is required');
        }

        return $workspaceId;
    }

    private function taskDurationSeconds(FleetTask $fleetTask): int
    {
        if ($fleetTask->started_at === null) {
            return 0;
        }

        return max(
            0,
            (int) $fleetTask->started_at->diffInSeconds($fleetTask->completed_at ?? now()),
        );
    }
}
