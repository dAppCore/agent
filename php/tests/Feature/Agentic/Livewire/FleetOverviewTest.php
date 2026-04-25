<?php

// SPDX-License-Identifier: EUPL-1.2

declare(strict_types=1);

use Core\Mod\Agentic\Models\FleetNode;
use Illuminate\Support\Facades\Blade;
use Livewire\Livewire;

use function Pest\Laravel\assertDatabaseHas;

uses(\Core\Mod\Agentic\Tests\Feature\Livewire\LivewireTestCase::class);

if (! function_exists('prepareAgenticLivewireHarness')) {
    function prepareAgenticLivewireHarness(): void
    {
        $base = sys_get_temp_dir().'/agentic-livewire-stubs';
        $componentPath = $base.'/components';
        $hubPath = $base.'/hub/admin/layouts';

        if (! is_dir($componentPath)) {
            mkdir($componentPath, 0777, true);
        }

        if (! is_dir($hubPath)) {
            mkdir($hubPath, 0777, true);
        }

        file_put_contents($hubPath.'/app.blade.php', '{{ $slot }}');

        $stubs = [
            'badge.blade.php' => '<span {{ $attributes }}>{{ $slot }}</span>',
            'button.blade.php' => '<button {{ $attributes }}>{{ $slot }}</button>',
            'card.blade.php' => '<div {{ $attributes }}>{{ $slot }}</div>',
            'heading.blade.php' => '<div {{ $attributes }}>{{ $slot }}</div>',
            'input.blade.php' => '<input {{ $attributes }} />',
            'select.blade.php' => '<select {{ $attributes }}>{{ $slot }}</select>',
            'text.blade.php' => '<div {{ $attributes }}>{{ $slot }}</div>',
            'textarea.blade.php' => '<textarea {{ $attributes }}>{{ $slot }}</textarea>',
        ];

        foreach ($stubs as $file => $contents) {
            file_put_contents($componentPath.'/'.$file, $contents);
        }

        Blade::anonymousComponentPath($componentPath, 'flux');
        app('view')->addNamespace('hub', $base.'/hub');
    }
}

if (! function_exists('loadAgenticLivewireComponent')) {
    function loadAgenticLivewireComponent(string $component): string
    {
        $phpRoot = dirname(__DIR__, 4);
        require_once $phpRoot."/Agentic/Livewire/{$component}.php";

        return "Core\\Mod\\Agentic\\Livewire\\{$component}";
    }
}

beforeEach(function (): void {
    prepareAgenticLivewireHarness();
    $this->actingAsHades();
});

it('wires fleet actions and flux blade controls', function (): void {
    $phpRoot = dirname(__DIR__, 4);
    $componentSource = file_get_contents($phpRoot.'/Agentic/Livewire/FleetOverview.php');
    $bladeSource = file_get_contents($phpRoot.'/resources/views/livewire/agentic/fleet-overview.blade.php');

    expect($componentSource)
        ->toContain('AssignTask')
        ->toContain('GetFleetStats')
        ->toContain('ListNodes');

    expect($bladeSource)
        ->toContain('<flux:card')
        ->toContain('wire:click="stageDispatch')
        ->toContain('wire:submit="dispatchTask"');
});

it('renders node list and stats for hades users', function (): void {
    $component = loadAgenticLivewireComponent('FleetOverview');
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
    $component = loadAgenticLivewireComponent('FleetOverview');
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
