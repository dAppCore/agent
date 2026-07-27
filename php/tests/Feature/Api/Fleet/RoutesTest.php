<?php

// SPDX-License-Identifier: EUPL-1.2

declare(strict_types=1);

// NOTE: updated for the fleet reconciliation — /v1/fleet/* runs on DispatchService
// over agent_registrations + dispatch_jobs. Flagged UNRUN: the framework test
// suite can't be installed here (forge offline); verify in CI.

use Core\Mod\Agentic\Controllers\Api\Fleet\FleetController;
use Core\Mod\Agentic\Models\AgentApiKey;
use Core\Mod\Agentic\Models\AgentRegistration;
use Core\Mod\Agentic\Models\DispatchJob;
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

test('fleet heartbeat route updates the agent status', function (): void {
    $workspace = createWorkspace();
    $key = fleetRouteKey($workspace);

    AgentRegistration::create([
        'workspace_id' => $workspace->id,
        'agent_id' => 'charon',
        'hostname' => 'charon',
        'platform' => 'linux',
        'status' => AgentRegistration::STATUS_OFFLINE,
    ]);

    $response = $this
        ->withHeader('Authorization', 'Bearer '.$key->plainTextKey)
        ->postJson('/v1/fleet/heartbeat', [
            'agent_id' => 'charon',
            'status' => AgentRegistration::STATUS_ONLINE,
            'compute_budget' => ['max_daily_hours' => 6],
        ]);

    $response
        ->assertOk()
        ->assertJsonPath('data.agent_id', 'charon')
        ->assertJsonPath('data.status', AgentRegistration::STATUS_ONLINE)
        ->assertJsonPath('data.compute_budget.max_daily_hours', 6);
});

test('fleet nodes route lists agents for the workspace', function (): void {
    $workspace = createWorkspace();
    $key = fleetRouteKey($workspace, [AgentApiKey::PERM_FLEET_READ]);

    AgentRegistration::create([
        'workspace_id' => $workspace->id,
        'agent_id' => 'clotho',
        'hostname' => 'clotho',
        'platform' => 'darwin',
        'status' => AgentRegistration::STATUS_ONLINE,
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

test('fleet dispatch route queues an unassigned job', function (): void {
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
        ->assertJsonPath('data.status', DispatchJob::STATUS_PENDING);

    expect(DispatchJob::query()->where('workspace_id', $workspace->id)->count())->toBe(1);
});

test('fleet stats route returns aggregate counters', function (): void {
    $workspace = createWorkspace();
    $key = fleetRouteKey($workspace, [AgentApiKey::PERM_FLEET_READ]);

    AgentRegistration::create([
        'workspace_id' => $workspace->id,
        'agent_id' => 'virgil',
        'hostname' => 'virgil',
        'platform' => 'linux',
        'status' => AgentRegistration::STATUS_ONLINE,
    ]);

    DispatchJob::create([
        'workspace_id' => $workspace->id,
        'repo' => 'core/agent',
        'task' => 'Summarise fleet throughput',
        'status' => DispatchJob::STATUS_COMPLETED,
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

test('fleet stream route emits sse frames for claimed jobs', function (): void {
    $workspace = createWorkspace();

    AgentRegistration::create([
        'workspace_id' => $workspace->id,
        'agent_id' => 'charon',
        'hostname' => 'charon',
        'platform' => 'linux',
        'status' => AgentRegistration::STATUS_ONLINE,
        'max_concurrent' => 1,
        'last_heartbeat_at' => now(),
    ]);

    $job = DispatchJob::create([
        'workspace_id' => $workspace->id,
        'repo' => 'core/app',
        'task' => 'Ship the stream alias',
        'status' => DispatchJob::STATUS_PENDING,
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
        ->and($output)->toContain('Ship the stream alias');

    $job->refresh();

    expect($job->status)->toBe(DispatchJob::STATUS_ASSIGNED)
        ->and($job->assigned_agent)->toBe('charon');
});
