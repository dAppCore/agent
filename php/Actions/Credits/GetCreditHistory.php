<?php

declare(strict_types=1);

namespace Core\Mod\Agentic\Actions\Credits;

use Core\Actions\Action;
use Core\Mod\Agentic\Models\CreditEntry;
use Core\Mod\Agentic\Models\FleetNode;
use Illuminate\Database\Eloquent\Collection;

class GetCreditHistory
{
    use Action;

    /**
     * @throws \InvalidArgumentException
     */
    public function handle(int $workspaceId, string $agentId, int $limit = 50): Collection
    {
        $node = FleetNode::query()
            ->where('workspace_id', $workspaceId)
            ->where('agent_id', $agentId)
            ->first();

        if (! $node) {
            throw new \InvalidArgumentException('Fleet node not found');
        }

        return CreditEntry::query()
            ->where('workspace_id', $workspaceId)
            ->where('fleet_node_id', $node->id)
            ->latest()
            ->limit($limit)
            ->get();
    }
}
