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

        $task = FleetTask::pendingForNode($node)->first();

        if (! $task) {
            return null;
        }

        $task->update([
            'status' => FleetTask::STATUS_IN_PROGRESS,
            'started_at' => $task->started_at ?? now(),
        ]);

        $node->update([
            'status' => FleetNode::STATUS_BUSY,
            'current_task_id' => $task->id,
        ]);

        return $task->fresh();
    }
}
