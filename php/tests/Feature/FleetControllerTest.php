<?php

declare(strict_types=1);

// NOTE: updated for the fleet reconciliation (FleetController now runs on
// DispatchService over agent_registrations + dispatch_jobs). Flagged UNRUN —
// the framework test suite can't be installed in the current environment
// (forge offline); verify in CI.

use Core\Mod\Agentic\Controllers\Api\FleetController;
use Core\Mod\Agentic\Models\AgentRegistration;
use Core\Mod\Agentic\Models\DispatchJob;
use Core\Tenant\Models\Workspace;
use Illuminate\Http\Request;

it('streams claimed dispatch jobs as SSE events', function () {
    $workspace = Workspace::factory()->create();

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
        'task' => 'Fix the failing tests',
        'status' => DispatchJob::STATUS_PENDING,
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

    expect($output)->toContain('event: ready')
        ->and($output)->toContain('"agent_id":"charon"')
        ->and($output)->toContain('event: task.assigned')
        ->and($output)->toContain('Fix the failing tests');

    $job->refresh();

    expect($job->status)->toBe(DispatchJob::STATUS_ASSIGNED)
        ->and($job->assigned_agent)->toBe('charon');
});
