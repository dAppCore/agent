<?php

declare(strict_types=1);

namespace Core\Mod\Agentic\Actions\Fleet;

use Core\Actions\Action;
use Core\Mod\Agentic\Models\FleetNode;
use Core\Mod\Agentic\Models\FleetTask;

class GetNextTask
{
    use Action;

    /**
     * @param  array<string, mixed>  $capabilities
     *
     * @throws \InvalidArgumentException
     */
    public function handle(int $workspaceId, string $agentId, array $capabilities = []): ?FleetTask
    {
        $node = FleetNode::query()
            ->where('workspace_id', $workspaceId)
            ->where('agent_id', $agentId)
            ->first();

        if (! $node) {
            throw new \InvalidArgumentException('Fleet node not found');
        }

        if (in_array($node->status, [FleetNode::STATUS_OFFLINE, FleetNode::STATUS_PAUSED], true)) {
            return null;
        }

        $task = FleetTask::pendingForNode($node)->first();

        if (! $task && ! $this->exceedsDailyBudget($node)) {
            $task = $this->selectQueuedTask($workspaceId, $node, $capabilities);
        }

        if (! $task) {
            return null;
        }

        $task->update(array_filter([
            'fleet_node_id' => $task->fleet_node_id ?? $node->id,
            'status' => FleetTask::STATUS_IN_PROGRESS,
            'started_at' => $task->started_at ?? now(),
        ], static fn (mixed $value): bool => $value !== null));

        $node->update([
            'status' => FleetNode::STATUS_BUSY,
            'current_task_id' => $task->id,
            'last_heartbeat_at' => now(),
        ]);

        return $task->fresh();
    }

    /**
     * @param  array<string, mixed>  $capabilities
     */
    private function selectQueuedTask(int $workspaceId, FleetNode $node, array $capabilities): ?FleetTask
    {
        $preferredRepo = $this->lastTouchedRepo($node);
        $nodeCapabilities = $this->normaliseCapabilities(array_merge(
            $node->capabilities ?? [],
            $capabilities,
        ));

        $tasks = FleetTask::query()
            ->where('workspace_id', $workspaceId)
            ->whereNull('fleet_node_id')
            ->whereIn('status', [FleetTask::STATUS_ASSIGNED, FleetTask::STATUS_QUEUED])
            ->get()
            ->filter(fn (FleetTask $fleetTask): bool => $this->matchesCapabilities($fleetTask, $nodeCapabilities))
            ->sortBy(fn (FleetTask $fleetTask): string => sprintf(
                '%d-%d-%010d-%010d',
                $this->priorityWeight($fleetTask),
                $preferredRepo !== null && $fleetTask->repo === $preferredRepo ? 0 : 1,
                $fleetTask->created_at?->getTimestamp() ?? 0,
                $fleetTask->id,
            ));

        $task = $tasks->first();

        return $task instanceof FleetTask ? $task : null;
    }

    private function exceedsDailyBudget(FleetNode $node): bool
    {
        $maxDailyHours = (float) ($node->compute_budget['max_daily_hours'] ?? 0);
        if ($maxDailyHours <= 0) {
            return false;
        }

        $usedSeconds = $node->tasks()
            ->whereDate('started_at', today())
            ->get()
            ->sum(fn (FleetTask $fleetTask): int => $this->taskDurationSeconds($fleetTask));

        return $usedSeconds >= (int) round($maxDailyHours * 3600);
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

    private function lastTouchedRepo(FleetNode $node): ?string
    {
        return $node->tasks()
            ->whereNotNull('repo')
            ->orderByDesc('completed_at')
            ->orderByDesc('updated_at')
            ->value('repo');
    }

    /**
     * @param  array<int, mixed>  $capabilities
     */
    private function normaliseCapabilities(array $capabilities): array
    {
        $normalised = [];
        foreach ($capabilities as $key => $value) {
            if (is_string($key) && $value) {
                $normalised[] = $key;
            }
            if (is_string($value) && $value !== '') {
                $normalised[] = $value;
            }
        }

        return array_values(array_unique($normalised));
    }

    /**
     * @param  array<int, string>  $nodeCapabilities
     */
    private function matchesCapabilities(FleetTask $fleetTask, array $nodeCapabilities): bool
    {
        $report = is_array($fleetTask->report) ? $fleetTask->report : [];
        $required = $this->normaliseCapabilities((array) ($report['required_capabilities'] ?? []));
        if ($required === []) {
            return true;
        }

        return array_diff($required, $nodeCapabilities) === [];
    }

    private function priorityWeight(FleetTask $fleetTask): int
    {
        $report = is_array($fleetTask->report) ? $fleetTask->report : [];
        $priority = strtoupper((string) ($report['priority'] ?? 'P2'));

        return match ($priority) {
            'P0' => 0,
            'P1' => 1,
            'P2' => 2,
            'P3' => 3,
            default => 4,
        };
    }
}
