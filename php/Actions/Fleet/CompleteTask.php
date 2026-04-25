<?php

declare(strict_types=1);

namespace Core\Mod\Agentic\Actions\Fleet;

use Core\Actions\Action;
use Core\Mod\Agentic\Actions\Credits\AwardCredits;
use Core\Mod\Agentic\Models\FleetNode;
use Core\Mod\Agentic\Models\FleetTask;

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
        $node = FleetNode::query()
            ->where('workspace_id', $workspaceId)
            ->where('agent_id', $agentId)
            ->first();

        $fleetTask = FleetTask::query()
            ->where('workspace_id', $workspaceId)
            ->find($taskId);

        if (! $node || ! $fleetTask) {
            throw new \InvalidArgumentException('Fleet task not found');
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

        $node->update([
            'status' => FleetNode::STATUS_ONLINE,
            'current_task_id' => null,
            'last_heartbeat_at' => now(),
        ]);

        $creditAmount = max(1, count($findings) + 1);
        AwardCredits::run($workspaceId, $agentId, 'fleet-task', $creditAmount, $node->id, 'Fleet task completed');

        return $fleetTask->fresh();
    }
}
