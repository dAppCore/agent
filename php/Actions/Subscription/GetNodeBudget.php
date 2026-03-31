<?php

declare(strict_types=1);

namespace Core\Mod\Agentic\Actions\Subscription;

use Core\Actions\Action;
use Core\Mod\Agentic\Models\FleetNode;

class GetNodeBudget
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

        return $node->compute_budget ?? [];
    }
}
