<?php

// SPDX-License-Identifier: EUPL-1.2

declare(strict_types=1);

use Core\Mod\Agentic\Controllers\Api\Fleet\FleetController;
use Core\Mod\Agentic\Models\AgentApiKey;
use Core\Mod\Agentic\Models\FleetNode;
use Core\Mod\Agentic\Models\FleetTask;
use Core\Tenant\Models\Workspace;
use Illuminate\Http\Request;

beforeEach(function (): void {
    require __DIR__.'/../../../../Routes/api.php';
});

function fleetRouteKey(
    Workspace $workspace,
    array $permissions = [AgentApiKey::PERM_FLEET_READ, AgentApiKey::PERM_FLEET_WRITE]
): AgentApiKey {
    return createApiKey($workspace, 'Fleet Route Key', $permissions);
}

test('fleet heartbeat route updates the node status', function (): void {
    $workspace = createWorkspace();
    $key = fleetRouteKey($workspace);

    FleetNode::create([
        'workspace_id' => $workspace->id,
        'agent_id' => 'charon',
        'platform' => 'linux',
        'status' => FleetNode::STATUS_OFFLINE,
    ]);

    $response = $this
        ->withHeader('Authorization', 'Bearer '.$key->plainTextKey)
        ->postJson('/v1/fleet/heartbeat', [
            'agent_id' => 'charon',
            'status' => FleetNode::STATUS_ONLINE,
            'compute_budget' => ['max_daily_hours' => 6],
        ]);

    $response
        ->assertOk()
        ->assertJsonPath('data.agent_id', 'charon')
        ->assertJsonPath('data.status', FleetNode::STATUS_ONLINE)
        ->assertJsonPath('data.compute_budget.max_daily_hours', 6);
});

test('fleet nodes route lists nodes for the workspace', function (): void {
    $workspace = createWorkspace();
    $key = fleetRouteKey($workspace, [AgentApiKey::PERM_FLEET_READ]);

    FleetNode::create([
        'workspace_id' => $workspace->id,
        'agent_id' => 'clotho',
        'platform' => 'darwin',
        'status' => FleetNode::STATUS_ONLINE,
    ]);

    $response = $this
        ->withHeader('Authorization', 'Bearer '.$key->plainTextKey)
        ->getJson('/v1/fleet/nodes');

    $response
        ->assertOk()
        ->assertJsonPath('total', 1)
        ->assertJsonPath('data.0.agent_id', 'clotho')
        ->assertJsonPath('data.0.platform', 'darwin');
});

test('fleet dispatch route queues an unassigned task', function (): void {
    $workspace = createWorkspace();
    $key = fleetRouteKey($workspace, [AgentApiKey::PERM_FLEET_WRITE]);

    $response = $this
        ->withHeader('Authorization', 'Bearer '.$key->plainTextKey)
        ->postJson('/v1/fleet/dispatch', [
            'repo' => 'dappco.re/go/agent',
            'task' => 'Implement the dispatch alias route',
            'branch' => 'dev',
        ]);

    $response
        ->assertCreated()
        ->assertJsonPath('data.repo', 'dappco.re/go/agent')
        ->assertJsonPath('data.status', FleetTask::STATUS_QUEUED);

    expect(FleetTask::query()->where('workspace_id', $workspace->id)->count())->toBe(1);
});

test('fleet stats route returns aggregate counters', function (): void {
    $workspace = createWorkspace();
    $key = fleetRouteKey($workspace, [AgentApiKey::PERM_FLEET_READ]);
    $node = FleetNode::create([
        'workspace_id' => $workspace->id,
        'agent_id' => 'virgil',
        'platform' => 'linux',
        'status' => FleetNode::STATUS_ONLINE,
    ]);

    FleetTask::create([
        'workspace_id' => $workspace->id,
        'fleet_node_id' => $node->id,
        'repo' => 'core/agent',
        'task' => 'Summarise fleet throughput',
        'status' => FleetTask::STATUS_COMPLETED,
        'findings' => [['severity' => 'high'], ['severity' => 'low']],
        'started_at' => now()->subHour(),
        'completed_at' => now(),
    ]);

    $response = $this
        ->withHeader('Authorization', 'Bearer '.$key->plainTextKey)
        ->getJson('/v1/fleet/stats');

    $response
        ->assertOk()
        ->assertJsonPath('data.nodes_online', 1)
        ->assertJsonPath('data.tasks_today', 1)
        ->assertJsonPath('data.repos_touched', 1)
        ->assertJsonPath('data.findings_total', 2);
});

test('fleet stream route emits sse frames for assigned tasks', function (): void {
    $workspace = createWorkspace();
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
        'task' => 'Ship the stream alias',
        'status' => FleetTask::STATUS_ASSIGNED,
    ]);

    $request = Request::create('/v1/fleet/stream', 'GET', [
        'agent_id' => 'charon',
        'limit' => 1,
        'poll_interval_ms' => 100,
    ]);
    $request->attributes->set('workspace_id', $workspace->id);

    $response = app(FleetController::class)->stream($request);

    ob_start();
    $response->sendContent();
    $output = ob_get_clean();

    expect($output)->toContain('event: ready')
        ->and($output)->toContain('"agent_id":"charon"')
        ->and($output)->toContain('event: task.assigned')
        ->and($output)->toContain('"repo":"core/app"')
        ->and($output)->toContain('"task":"Ship the stream alias"');

    $task->refresh();
    $node->refresh();

    expect($task->status)->toBe(FleetTask::STATUS_IN_PROGRESS)
        ->and($node->status)->toBe(FleetNode::STATUS_BUSY)
        ->and($node->current_task_id)->toBe($task->id);
});
