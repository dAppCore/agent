<?php

declare(strict_types=1);

namespace Core\Mod\Agentic\Actions\Sync;

use Core\Actions\Action;
use Core\Mod\Agentic\Models\FleetNode;
use Core\Mod\Agentic\Models\SyncRecord;

class GetAgentSyncStatus
{
    use Action;

    /**
     * @return array<string, mixed>
     *
     * @throws \InvalidArgumentException
     */
    public function handle(int $workspaceId, string $agentId): array
    {
        $node = FleetNode::query()
            ->where('workspace_id', $workspaceId)
            ->where('agent_id', $agentId)
            ->first();

        if (! $node) {
            throw new \InvalidArgumentException('Fleet node not found');
        }

        $lastPush = SyncRecord::query()
            ->where('fleet_node_id', $node->id)
            ->where('direction', 'push')
            ->latest('synced_at')
            ->first();

        $lastPull = SyncRecord::query()
            ->where('fleet_node_id', $node->id)
            ->where('direction', 'pull')
            ->latest('synced_at')
            ->first();

        return [
            'agent_id' => $node->agent_id,
            'status' => $node->status,
            'last_push_at' => $lastPush?->synced_at?->toIso8601String(),
            'last_pull_at' => $lastPull?->synced_at?->toIso8601String(),
        ];
    }
}
