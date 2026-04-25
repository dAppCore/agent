<?php

// SPDX-License-Identifier: EUPL-1.2

declare(strict_types=1);

namespace Core\Mod\Agentic\Actions\Credits;

use Core\Actions\Action;
use Core\Mod\Agentic\Models\CreditEntry;
use Core\Mod\Agentic\Models\FleetNode;
use Illuminate\Database\Eloquent\Builder;
use Illuminate\Database\Eloquent\Collection;

class GetCreditHistory
{
    use Action;

    /**
     * @throws \InvalidArgumentException
     */
    public function handle(int $workspaceId, string $agentId, int $limit = 50): Collection
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

        return $query
            ->latest()
            ->limit($limit)
            ->get();
    }
}
