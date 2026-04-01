<?php

declare(strict_types=1);

use Core\Mod\Agentic\Controllers\Api\FleetController;
use Core\Mod\Agentic\Models\FleetNode;
use Core\Mod\Agentic\Models\FleetTask;
use Core\Tenant\Models\Workspace;
use Illuminate\Http\Request;

it('streams assigned fleet tasks as SSE events', function () {
    $workspace = Workspace::factory()->create();
    $node = FleetNode::create([
        'workspace_id' => $workspace->id,
        'agent_id' => 'charon',
        'platform' => 'linux',
        'status' => FleetNode::STATUS_ONLINE,
    ]);

    $task = FleetTask::create([
        'workspace_id' => $workspace->id,
        'fleet_node_id' => $node->id,
        'repo' => 'core/app',
        'task' => 'Fix the failing tests',
        'status' => FleetTask::STATUS_ASSIGNED,
    ]);

    $request = Request::create('/v1/fleet/events', 'GET', [
        'agent_id' => 'charon',
        'limit' => 1,
        'poll_interval_ms' => 100,
    ]);
    $request->attributes->set('workspace_id', $workspace->id);

    $response = app(FleetController::class)->events($request);

    ob_start();
    $response->sendContent();
    $output = ob_get_clean();

    expect($output)->toContain("event: ready")
        ->and($output)->toContain('"agent_id":"charon"')
        ->and($output)->toContain("event: task.assigned")
        ->and($output)->toContain('"repo":"core/app"')
        ->and($output)->toContain('"task":"Fix the failing tests"');

    $task->refresh();
    $node->refresh();

    expect($task->status)->toBe(FleetTask::STATUS_IN_PROGRESS)
        ->and($node->status)->toBe(FleetNode::STATUS_BUSY)
        ->and($node->current_task_id)->toBe($task->id);
});
