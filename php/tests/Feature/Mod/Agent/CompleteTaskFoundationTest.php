<?php

// SPDX-License-Identifier: EUPL-1.2

declare(strict_types=1);

use Core\Mod\Agentic\Actions\Fleet\CompleteTask;
use Core\Mod\Agentic\Models\CreditEntry;
use Core\Mod\Agentic\Models\FleetNode;
use Core\Mod\Agentic\Models\FleetTask;

test('agent foundation complete task is atomic and idempotent for credits', function (): void {
    $workspace = createWorkspace();

    $node = FleetNode::query()->create([
        'workspace_id' => $workspace->id,
        'agent_id' => 'charon',
        'platform' => 'linux',
        'status' => FleetNode::STATUS_BUSY,
        'registered_at' => now()->subMinutes(10),
        'last_heartbeat_at' => now()->subMinute(),
    ]);

    $task = FleetTask::query()->create([
        'workspace_id' => $workspace->id,
        'fleet_node_id' => $node->id,
        'repo' => 'dappco.re/go/agent',
        'branch' => 'dev',
        'task' => 'Complete the foundation slice',
        'status' => FleetTask::STATUS_IN_PROGRESS,
        'started_at' => now()->subMinutes(5),
    ]);

    $node->update(['current_task_id' => $task->id]);

    $completed = CompleteTask::run(
        $workspace->id,
        'charon',
        $task->id,
        ['status' => 'completed'],
        [['severity' => 'medium']],
        ['files_changed' => 3],
        ['summary' => 'Foundation delivered'],
    );

    CompleteTask::run(
        $workspace->id,
        'charon',
        $task->id,
        ['status' => 'completed'],
        [['severity' => 'medium']],
        ['files_changed' => 3],
        ['summary' => 'Foundation delivered'],
    );

    $creditEntry = CreditEntry::query()
        ->where('workspace_id', $workspace->id)
        ->where('fleet_node_id', $node->id)
        ->where('fleet_task_id', $task->id)
        ->first();

    expect($completed->status)->toBe(FleetTask::STATUS_COMPLETED)
        ->and($node->fresh()->status)->toBe(FleetNode::STATUS_ONLINE)
        ->and($node->fresh()->current_task_id)->toBeNull()
        ->and($creditEntry)->not->toBeNull()
        ->and($creditEntry?->agent_id)->toBe('charon')
        ->and(CreditEntry::query()
            ->where('workspace_id', $workspace->id)
            ->where('fleet_node_id', $node->id)
            ->where('fleet_task_id', $task->id)
            ->count())->toBe(1);
});
