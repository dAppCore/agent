<?php

declare(strict_types=1);

namespace Core\Mod\Agentic\Actions\Subscription;

use Core\Actions\Action;
use Core\Mod\Agentic\Models\FleetNode;

class UpdateBudget
{
    use Action;

    /**
     * @param  array<string, mixed>  $limits
     * @return array<string, mixed>
     *
     * @throws \InvalidArgumentException
     */
    public function handle(int $workspaceId, string $agentId, array $limits): array
    {
        $node = FleetNode::query()
            ->where('workspace_id', $workspaceId)
            ->where('agent_id', $agentId)
            ->first();

        if (! $node) {
            throw new \InvalidArgumentException('Fleet node not found');
        }

        $node->update([
            'compute_budget' => array_merge($node->compute_budget ?? [], $limits),
            'last_heartbeat_at' => now(),
        ]);

        return $node->fresh()->compute_budget ?? [];
    }
}
