<?php

declare(strict_types=1);

namespace Core\Mod\Agentic\Actions\Fleet;

use Core\Actions\Action;
use Core\Mod\Agentic\Models\FleetNode;

class DeregisterNode
{
    use Action;

    /**
     * @throws \InvalidArgumentException
     */
    public function handle(int $workspaceId, string $agentId): bool
    {
        $node = FleetNode::query()
            ->where('workspace_id', $workspaceId)
            ->where('agent_id', $agentId)
            ->first();

        if (! $node) {
            throw new \InvalidArgumentException('Fleet node not found');
        }

        $node->update([
            'status' => FleetNode::STATUS_OFFLINE,
            'current_task_id' => null,
            'last_heartbeat_at' => now(),
        ]);

        return true;
    }
}
