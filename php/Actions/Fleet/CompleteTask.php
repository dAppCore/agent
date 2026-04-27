<?php

// SPDX-License-Identifier: EUPL-1.2

declare(strict_types=1);

namespace Core\Mod\Agentic\Actions\Fleet;

use Core\Actions\Action;
use Core\Mod\Agentic\Actions\Credits\AwardCredits;
use Core\Mod\Agentic\Models\FleetNode;
use Core\Mod\Agentic\Models\FleetTask;
use Illuminate\Support\Facades\DB;

/**
 * Fleet tasks intentionally do not create AgentSession records. AgentSession tracks interactive,
 * replayable, handoff-capable work with a work_log and artefact history; fleet tasks are atomic
 * assign→complete events with no in-between state to replay. If a fleet task's work requires
 * session semantics, the agent executing the task should start an AgentSession itself via
 * AgentSessionService.
 */
class CompleteTask
{
    use Action;

    /**
     * @param  array<string, mixed>  $result
     * @param  array<int, mixed>  $findings
     * @param  array<string, mixed>  $changes
     * @param  array<string, mixed>  $report
     *
     * @throws \InvalidArgumentException
     */
    public function handle(
        int $workspaceId,
        string $agentId,
        int $taskId,
        array $result = [],
        array $findings = [],
        array $changes = [],
        array $report = []
    ): FleetTask {
        return DB::transaction(function () use (
            $workspaceId,
            $agentId,
            $taskId,
            $result,
            $findings,
            $changes,
            $report,
        ): FleetTask {
            $node = FleetNode::query()
                ->where('workspace_id', $workspaceId)
                ->where('agent_id', $agentId)
                ->lockForUpdate()
                ->first();

            $fleetTask = FleetTask::query()
                ->where('workspace_id', $workspaceId)
                ->lockForUpdate()
                ->find($taskId);

            if (! $node instanceof FleetNode || ! $fleetTask instanceof FleetTask) {
                throw new \InvalidArgumentException('Fleet task not found');
            }

            if ($fleetTask->fleet_node_id !== null && $fleetTask->fleet_node_id !== $node->id) {
                throw new \InvalidArgumentException('Fleet task does not belong to this node');
            }

            $status = ($result['status'] ?? '') === 'failed'
                ? FleetTask::STATUS_FAILED
                : FleetTask::STATUS_COMPLETED;

            $fleetTask->update([
                'status' => $status,
                'result' => $result,
                'findings' => $findings,
                'changes' => $changes,
                'report' => $report,
                'completed_at' => now(),
            ]);

            $creditAmount = max(1, count($findings) + 1);
            AwardCredits::run(
                $workspaceId,
                $agentId,
                'fleet-task',
                $creditAmount,
                $node->id,
                'Fleet task completed',
                $fleetTask->id,
            );

            $nodeUpdate = [
                'last_heartbeat_at' => now(),
            ];

            if ($node->current_task_id === null || $node->current_task_id === $fleetTask->id) {
                $nodeUpdate['status'] = FleetNode::STATUS_ONLINE;
                $nodeUpdate['current_task_id'] = null;
            }

            $node->update($nodeUpdate);

            return $fleetTask->fresh();
        });
    }
}
