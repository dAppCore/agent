<?php

// SPDX-License-Identifier: EUPL-1.2

declare(strict_types=1);

namespace Core\Mod\Agentic\Tests\Feature\Agentic\Livewire;

use Core\Mod\Agentic\Models\FleetNode;
use Core\Mod\Agentic\Tests\Feature\Livewire\LivewireTestCase;
use Livewire\Livewire;

/**
 * Class-style for the same reason as BrainExplorerTest: a file-level
 * uses(LivewireTestCase::class) collides with the directory-wide
 * uses(Tests\TestCase::class) that Pest.php applies to Feature.
 */
class FleetOverviewTest extends LivewireTestCase
{
    protected function setUp(): void
    {
        parent::setUp();

        $this->actingAsHades();
    }

    public function test_wires_fleet_actions_and_flux_blade_controls(): void
    {
        $this->assertFluxComponentWiring(
            'FleetOverview',
            'fleet-overview',
            ['AssignTask', 'GetFleetStats', 'ListNodes'],
            ['<flux:card', 'wire:click="stageDispatch', 'wire:submit="dispatchTask"'],
        );
    }

    public function test_renders_node_list_and_stats_for_hades_users(): void
    {
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
    }

    public function test_dispatches_a_task_to_the_selected_node(): void
    {
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

        $this->assertDatabaseHas('fleet_tasks', [
            'workspace_id' => $workspace->id,
            'repo' => 'dAppCore/core-agent',
            'branch' => 'dev',
            'template' => 'triage',
            'agent_model' => 'gpt-5.5',
            'status' => 'assigned',
        ]);

        $this->assertDatabaseHas('fleet_nodes', [
            'workspace_id' => $workspace->id,
            'agent_id' => 'alpha',
            'status' => FleetNode::STATUS_BUSY,
        ]);
    }
}
