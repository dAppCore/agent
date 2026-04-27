<?php

// SPDX-License-Identifier: EUPL-1.2

declare(strict_types=1);

use Core\Mod\Agentic\Models\FleetNode;
use Livewire\Livewire;

use function Pest\Laravel\assertDatabaseHas;

uses(\Core\Mod\Agentic\Tests\Feature\Livewire\LivewireTestCase::class);

beforeEach(function (): void {
    $this->actingAsHades();
});

it('wires fleet actions and flux blade controls', function (): void {
    $this->assertFluxComponentWiring(
        'FleetOverview',
        'fleet-overview',
        ['AssignTask', 'GetFleetStats', 'ListNodes'],
        ['<flux:card', 'wire:click="stageDispatch', 'wire:submit="dispatchTask"'],
    );
});

it('renders node list and stats for hades users', function (): void {
    $component = $this->livewireComponent('FleetOverview');
    $workspace = createWorkspace();

    FleetNode::query()->create([
        'workspace_id' => $workspace->id,
        'agent_id' => 'alpha',
        'platform' => 'darwin',
        'models' => ['gpt-5.5'],
        'status' => FleetNode::STATUS_ONLINE,
        'registered_at' => now(),
        'last_heartbeat_at' => now(),
    ]);

    FleetNode::query()->create([
        'workspace_id' => $workspace->id,
        'agent_id' => 'beta',
        'platform' => 'linux',
        'models' => ['gpt-5.4-mini'],
        'status' => FleetNode::STATUS_BUSY,
        'registered_at' => now(),
        'last_heartbeat_at' => now(),
    ]);

    Livewire::test($component, ['workspaceId' => $workspace->id])
        ->assertSee('Fleet Overview')
        ->assertSee('Dispatch Task')
        ->assertSee('alpha')
        ->assertSee('beta');
});

it('dispatches a task to the selected node', function (): void {
    $component = $this->livewireComponent('FleetOverview');
    $workspace = createWorkspace();

    FleetNode::query()->create([
        'workspace_id' => $workspace->id,
        'agent_id' => 'alpha',
        'platform' => 'darwin',
        'models' => ['gpt-5.5'],
        'status' => FleetNode::STATUS_ONLINE,
        'registered_at' => now(),
        'last_heartbeat_at' => now(),
    ]);

    Livewire::test($component, ['workspaceId' => $workspace->id])
        ->set('dispatchAgentId', 'alpha')
        ->set('dispatchRepo', 'dAppCore/core-agent')
        ->set('dispatchBranch', 'dev')
        ->set('dispatchTemplate', 'triage')
        ->set('dispatchModel', 'gpt-5.5')
        ->set('dispatchTask', 'Review the dispatch backlog and prepare the next assignment.')
        ->call('dispatchTask')
        ->assertHasNoErrors();

    assertDatabaseHas('fleet_tasks', [
        'workspace_id' => $workspace->id,
        'repo' => 'dAppCore/core-agent',
        'branch' => 'dev',
        'template' => 'triage',
        'agent_model' => 'gpt-5.5',
        'status' => 'assigned',
    ]);

    assertDatabaseHas('fleet_nodes', [
        'workspace_id' => $workspace->id,
        'agent_id' => 'alpha',
        'status' => FleetNode::STATUS_BUSY,
    ]);
});
