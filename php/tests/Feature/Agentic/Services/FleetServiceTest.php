<?php

// SPDX-License-Identifier: EUPL-1.2

declare(strict_types=1);

use Core\Mod\Agentic\Data\FleetStats;
use Core\Mod\Agentic\Models\FleetNode;
use Core\Mod\Agentic\Models\FleetTask;
use Core\Mod\Agentic\Services\FleetService;

use function Pest\Laravel\assertDatabaseHas;

if (! function_exists('loadAgenticPhpClass')) {
    function loadAgenticPhpClass(string $relativePath): void
    {
        $phpRoot = dirname(__DIR__, 4);
        require_once $phpRoot.'/'.$relativePath;
    }
}

beforeEach(function (): void {
    loadAgenticPhpClass('Agentic/Data/FleetStats.php');
    loadAgenticPhpClass('Agentic/Services/FleetService.php');
});

test('FleetService_register_Good_registers_heartbeats_and_reports_workspace_stats', function (): void {
    $workspace = createWorkspace();
    $service = new FleetService();

    $registered = $service->register([
        'workspace_id' => $workspace->id,
        'agent_id' => 'alpha',
        'platform' => 'darwin',
        'models' => ['gpt-5.5'],
        'capabilities' => ['dispatch' => true],
    ]);

    $heartbeat = $service->heartbeat([
        'workspace_id' => $workspace->id,
        'agent_id' => 'alpha',
        'status' => FleetNode::STATUS_BUSY,
        'compute_budget' => ['max_daily_hours' => 4],
    ]);

    $task = $service->dispatch($workspace->id, [
        'agent_id' => 'alpha',
        'repo' => 'dAppCore/core-agent',
        'branch' => 'dev',
        'task' => 'Review the next queue item and prepare an assignment.',
    ]);

    $stats = $service->stats($workspace->id);

    expect($registered->status)->toBe(FleetNode::STATUS_ONLINE)
        ->and($heartbeat->status)->toBe(FleetNode::STATUS_BUSY)
        ->and($task->status)->toBe(FleetTask::STATUS_ASSIGNED)
        ->and($stats)->toBeInstanceOf(FleetStats::class)
        ->and($stats->nodesOnline)->toBe(1)
        ->and($stats->tasksToday)->toBe(1);

    assertDatabaseHas('fleet_nodes', [
        'workspace_id' => $workspace->id,
        'agent_id' => 'alpha',
        'status' => FleetNode::STATUS_BUSY,
    ]);

    assertDatabaseHas('fleet_tasks', [
        'workspace_id' => $workspace->id,
        'repo' => 'dAppCore/core-agent',
        'status' => FleetTask::STATUS_ASSIGNED,
    ]);
});

test('FleetService_dispatch_Bad_rejects_missing_repo_or_task', function (): void {
    $workspace = createWorkspace();
    $service = new FleetService();

    expect(fn () => $service->dispatch($workspace->id, [
        'repo' => '',
        'task' => '   ',
    ]))->toThrow(InvalidArgumentException::class, 'repo and task are required');
});

test('FleetService_dispatch_Ugly_queues_unassigned_work_and_marks_stale_nodes_unhealthy', function (): void {
    $workspace = createWorkspace();
    $service = new FleetService();

    $node = FleetNode::query()->create([
        'workspace_id' => $workspace->id,
        'agent_id' => 'beta',
        'platform' => 'linux',
        'status' => FleetNode::STATUS_ONLINE,
        'last_heartbeat_at' => now()->subMinutes(10),
        'registered_at' => now()->subMinutes(10),
    ]);

    $queued = $service->dispatch($workspace->id, [
        'repo' => 'dAppCore/core-agent',
        'task' => 'Pick this up when capacity is available.',
        'report' => ['priority' => 'P1'],
    ]);

    $health = $service->health($node);

    expect($queued->status)->toBe(FleetTask::STATUS_QUEUED)
        ->and($health['agent_id'])->toBe('beta')
        ->and($health['is_stale'])->toBeTrue()
        ->and($health['is_online'])->toBeTrue();

    assertDatabaseHas('fleet_tasks', [
        'workspace_id' => $workspace->id,
        'repo' => 'dAppCore/core-agent',
        'status' => FleetTask::STATUS_QUEUED,
    ]);
});
