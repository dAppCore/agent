<?php

// SPDX-License-Identifier: EUPL-1.2

declare(strict_types=1);

namespace Core\Mod\Agentic\Actions\Credits;

use Core\Actions\Action;
use Core\Mod\Agentic\Models\CreditEntry;
use Core\Mod\Agentic\Models\FleetNode;
use Core\Mod\Agentic\Models\FleetTask;
use Illuminate\Support\Facades\DB;

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
        ?string $description = null,
        ?int $fleetTaskId = null,
    ): CreditEntry {
        if ($agentId === '' || $taskType === '' || $amount === 0) {
            throw new \InvalidArgumentException('agent_id, task_type, and non-zero amount are required');
        }

        return DB::transaction(function () use (
            $workspaceId,
            $agentId,
            $taskType,
            $amount,
            $fleetNodeId,
            $description,
            $fleetTaskId,
        ): CreditEntry {
            $node = $fleetNodeId !== null
                ? FleetNode::query()->where('workspace_id', $workspaceId)->lockForUpdate()->find($fleetNodeId)
                : FleetNode::query()
                    ->where('workspace_id', $workspaceId)
                    ->where('agent_id', $agentId)
                    ->lockForUpdate()
                    ->first();

            if (! $node instanceof FleetNode) {
                throw new \InvalidArgumentException('Fleet node not found');
            }

            if ($fleetTaskId !== null) {
                $fleetTask = FleetTask::query()
                    ->where('workspace_id', $workspaceId)
                    ->lockForUpdate()
                    ->find($fleetTaskId);

                if (! $fleetTask instanceof FleetTask) {
                    throw new \InvalidArgumentException('Fleet task not found');
                }

                if ($fleetTask->fleet_node_id !== null && $fleetTask->fleet_node_id !== $node->id) {
                    throw new \InvalidArgumentException('Fleet task does not belong to the node being credited');
                }

                $existing = CreditEntry::query()
                    ->where('workspace_id', $workspaceId)
                    ->where('fleet_node_id', $node->id)
                    ->where('fleet_task_id', $fleetTaskId)
                    ->first();

                if ($existing instanceof CreditEntry) {
                    return $existing;
                }
            }

            $previousBalance = (int) CreditEntry::query()
                ->where('workspace_id', $workspaceId)
                ->where('agent_id', $node->agent_id)
                ->latest('id')
                ->value('balance_after');

            return CreditEntry::query()->create([
                'workspace_id' => $workspaceId,
                'fleet_node_id' => $node->id,
                'fleet_task_id' => $fleetTaskId,
                'agent_id' => $node->agent_id,
                'task_type' => $taskType,
                'amount' => $amount,
                'balance_after' => $previousBalance + $amount,
                'description' => $description,
            ]);
        });
    }
}
