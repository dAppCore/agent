<?php

// SPDX-License-Identifier: EUPL-1.2

declare(strict_types=1);

namespace Core\Mod\Agentic\Actions\Credits;

use Core\Actions\Action;
use Core\Mod\Agentic\Models\CreditEntry;
use Core\Mod\Agentic\Models\FleetNode;
use Illuminate\Database\Eloquent\Builder;

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
        $nodeId = FleetNode::query()
            ->where('workspace_id', $workspaceId)
            ->where('agent_id', $agentId)
            ->value('id');

        $query = CreditEntry::query()
            ->where('workspace_id', $workspaceId)
            ->where(function (Builder $builder) use ($agentId, $nodeId): void {
                $builder->where('agent_id', $agentId);

                if ($nodeId !== null) {
                    $builder->orWhere(function (Builder $legacy) use ($nodeId): void {
                        $legacy->whereNull('agent_id')
                            ->where('fleet_node_id', $nodeId);
                    });
                }
            });

        if ($nodeId === null && ! $query->exists()) {
            throw new \InvalidArgumentException('Fleet node not found');
        }

        $balance = (int) (clone $query)
            ->latest('id')
            ->value('balance_after');
        $totalEarned = (int) (clone $query)->where('amount', '>', 0)->sum('amount');
        $totalSpent = (int) abs((int) (clone $query)->where('amount', '<', 0)->sum('amount'));
        $entries = (clone $query)->count();

        return [
            'agent_id' => $agentId,
            'balance' => $balance,
            'total_earned' => $totalEarned,
            'total_spent' => $totalSpent,
            'entries' => $entries,
        ];
    }
}
