<?php

// SPDX-License-Identifier: EUPL-1.2

declare(strict_types=1);

use Core\Mod\Agentic\Actions\Credits\AwardCredits;
use Core\Mod\Agentic\Actions\Credits\GetBalance;
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

it('wires credit actions and flux blade controls', function (): void {
    $phpRoot = dirname(__DIR__, 4);
    $componentSource = file_get_contents($phpRoot.'/Agentic/Livewire/CreditLedger.php');
    $bladeSource = file_get_contents($phpRoot.'/resources/views/livewire/agentic/credit-ledger.blade.php');

    expect($componentSource)
        ->toContain('AwardCredits')
        ->toContain('GetBalance')
        ->toContain('GetCreditHistory');

    expect($bladeSource)
        ->toContain('<flux:card')
        ->toContain('wire:click="deductCredits"')
        ->toContain('wire:click="refundCredits"');
});

it('renders balance and transaction history for the selected agent', function (): void {
    $component = loadAgenticLivewireComponent('CreditLedger');
    $workspace = createWorkspace();

    FleetNode::query()->create([
        'workspace_id' => $workspace->id,
        'agent_id' => 'alpha',
        'platform' => 'darwin',
        'status' => FleetNode::STATUS_ONLINE,
        'registered_at' => now(),
        'last_heartbeat_at' => now(),
    ]);

    AwardCredits::run($workspace->id, 'alpha', 'manual-refund', 5, null, 'Initial award');

    Livewire::test($component, ['workspaceId' => $workspace->id])
        ->assertSee('Credit Ledger')
        ->assertSee('alpha')
        ->assertSee('Initial award')
        ->assertSee('5');
});

it('refunds and deducts credits through the ledger actions', function (): void {
    $component = loadAgenticLivewireComponent('CreditLedger');
    $workspace = createWorkspace();

    FleetNode::query()->create([
        'workspace_id' => $workspace->id,
        'agent_id' => 'alpha',
        'platform' => 'darwin',
        'status' => FleetNode::STATUS_ONLINE,
        'registered_at' => now(),
        'last_heartbeat_at' => now(),
    ]);

    Livewire::test($component, ['workspaceId' => $workspace->id])
        ->set('selectedAgentId', 'alpha')
        ->set('adjustmentAmount', 3)
        ->set('adjustmentReason', 'Manual refund')
        ->call('refundCredits')
        ->assertHasNoErrors()
        ->set('adjustmentAmount', 2)
        ->set('adjustmentReason', 'Manual deduction')
        ->call('deductCredits')
        ->assertHasNoErrors();

    assertDatabaseHas('credit_entries', [
        'workspace_id' => $workspace->id,
        'task_type' => 'manual-refund',
        'amount' => 3,
        'description' => 'Manual refund',
    ]);

    assertDatabaseHas('credit_entries', [
        'workspace_id' => $workspace->id,
        'task_type' => 'manual-deduction',
        'amount' => -2,
        'description' => 'Manual deduction',
    ]);

    expect(GetBalance::run($workspace->id, 'alpha')['balance'])->toBe(1);
});
