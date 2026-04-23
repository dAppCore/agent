<?php

declare(strict_types=1);

namespace Core\Mod\Agentic\Actions\Fleet;

use Core\Actions\Action;
use Core\Mod\Agentic\Models\FleetNode;
use Core\Mod\Agentic\Models\FleetTask;

/**
 * Fleet tasks intentionally do not create AgentSession records. AgentSession tracks interactive,
 * replayable, handoff-capable work with a work_log and artefact history; fleet tasks are atomic
 * assign→complete events with no in-between state to replay. If a fleet task's work requires
 * session semantics, the agent executing the task should start an AgentSession itself via
 * AgentSessionService.
 */
class AssignTask
{
    use Action;

    /**
     * @throws \InvalidArgumentException
     */
    public function handle(
        int $workspaceId,
        string $agentId,
        string $task,
        string $repo,
        ?string $template = null,
        ?string $branch = null,
        ?string $agentModel = null
    ): FleetTask {
        $node = FleetNode::query()
            ->where('workspace_id', $workspaceId)
            ->where('agent_id', $agentId)
            ->first();

        if (! $node) {
            throw new \InvalidArgumentException('Fleet node not found');
        }

        if ($task === '' || $repo === '') {
            throw new \InvalidArgumentException('repo and task are required');
        }

        $fleetTask = FleetTask::create([
            'workspace_id' => $workspaceId,
            'fleet_node_id' => $node->id,
            'repo' => $repo,
            'branch' => $branch,
            'task' => $task,
            'template' => $template,
            'agent_model' => $agentModel,
            'status' => FleetTask::STATUS_ASSIGNED,
        ]);

        $node->update([
            'status' => FleetNode::STATUS_BUSY,
            'current_task_id' => $fleetTask->id,
        ]);

        return $fleetTask->fresh();
    }
}
