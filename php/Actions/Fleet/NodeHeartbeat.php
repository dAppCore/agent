<?php

declare(strict_types=1);

namespace Core\Mod\Agentic\Actions\Fleet;

use Core\Actions\Action;
use Core\Mod\Agentic\Models\FleetNode;

class NodeHeartbeat
{
    use Action;

    /**
     * @param  array<string, mixed>  $computeBudget
     *
     * @throws \InvalidArgumentException
     */
    public function handle(int $workspaceId, string $agentId, string $status, array $computeBudget = []): FleetNode
    {
        $node = FleetNode::query()
            ->where('workspace_id', $workspaceId)
            ->where('agent_id', $agentId)
            ->first();

        if (! $node) {
            throw new \InvalidArgumentException('Fleet node not found');
        }

        $node->update([
            'status' => $status !== '' ? $status : $node->status,
            'compute_budget' => $computeBudget !== [] ? $computeBudget : $node->compute_budget,
            'last_heartbeat_at' => now(),
        ]);

        return $node->fresh();
    }
}
