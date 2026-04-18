<?php

declare(strict_types=1);

namespace Core\Mod\Agentic\Actions\Credits;

use Core\Actions\Action;
use Core\Mod\Agentic\Models\CreditEntry;
use Core\Mod\Agentic\Models\FleetNode;

class GetBalance
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

        $balance = (int) CreditEntry::query()
            ->where('workspace_id', $workspaceId)
            ->where('fleet_node_id', $node->id)
            ->latest('id')
            ->value('balance_after');

        return [
            'agent_id' => $agentId,
            'balance' => $balance,
            'entries' => CreditEntry::query()
                ->where('workspace_id', $workspaceId)
                ->where('fleet_node_id', $node->id)
                ->count(),
        ];
    }
}
