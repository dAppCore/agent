<?php

declare(strict_types=1);

namespace Core\Mod\Agentic\Actions\Credits;

use Core\Actions\Action;
use Core\Mod\Agentic\Models\CreditEntry;
use Core\Mod\Agentic\Models\FleetNode;

class AwardCredits
{
    use Action;

    /**
     * @throws \InvalidArgumentException
     */
    public function handle(
        int $workspaceId,
        string $agentId,
        string $taskType,
        int $amount,
        ?int $fleetNodeId = null,
        ?string $description = null
    ): CreditEntry {
        if ($agentId === '' || $taskType === '' || $amount === 0) {
            throw new \InvalidArgumentException('agent_id, task_type, and non-zero amount are required');
        }

        $node = $fleetNodeId !== null
            ? FleetNode::query()->where('workspace_id', $workspaceId)->find($fleetNodeId)
            : FleetNode::query()->where('workspace_id', $workspaceId)->where('agent_id', $agentId)->first();

        if (! $node) {
            throw new \InvalidArgumentException('Fleet node not found');
        }

        $previousBalance = (int) CreditEntry::query()
            ->where('workspace_id', $workspaceId)
            ->where('fleet_node_id', $node->id)
            ->latest('id')
            ->value('balance_after');

        return CreditEntry::create([
            'workspace_id' => $workspaceId,
            'fleet_node_id' => $node->id,
            'task_type' => $taskType,
            'amount' => $amount,
            'balance_after' => $previousBalance + $amount,
            'description' => $description,
        ]);
    }
}
