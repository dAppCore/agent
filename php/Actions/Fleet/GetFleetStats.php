<?php

declare(strict_types=1);

namespace Core\Mod\Agentic\Actions\Fleet;

use Core\Actions\Action;
use Core\Mod\Agentic\Models\FleetNode;
use Core\Mod\Agentic\Models\FleetTask;

class GetFleetStats
{
    use Action;

    /**
     * @return array<string, int>
     */
    public function handle(int $workspaceId): array
    {
        $nodes = FleetNode::query()->where('workspace_id', $workspaceId);
        $tasks = FleetTask::query()->where('workspace_id', $workspaceId);

        return [
            'nodes_online' => (clone $nodes)->online()->count(),
            'tasks_today' => (clone $tasks)->whereDate('created_at', today())->count(),
            'tasks_week' => (clone $tasks)->where('created_at', '>=', now()->subDays(7))->count(),
            'repos_touched' => (clone $tasks)->distinct('repo')->count('repo'),
            'findings_total' => (clone $tasks)->get()->sum(static fn (FleetTask $task) => count($task->findings ?? [])),
            'compute_hours' => 0,
        ];
    }
}
