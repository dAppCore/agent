<?php

// SPDX-License-Identifier: EUPL-1.2

declare(strict_types=1);

use Core\Mod\Agentic\Actions\Credits\AwardCredits;
use Core\Mod\Agentic\Actions\Credits\GetBalance;
use Core\Mod\Agentic\Models\FleetNode;
use Livewire\Livewire;

use function Pest\Laravel\assertDatabaseHas;

uses(\Core\Mod\Agentic\Tests\Feature\Livewire\LivewireTestCase::class);

beforeEach(function (): void {
    $this->actingAsHades();
});

it('wires credit actions and flux blade controls', function (): void {
    $this->assertFluxComponentWiring(
        'CreditLedger',
        'credit-ledger',
        ['AwardCredits', 'GetBalance', 'GetCreditHistory'],
        ['<flux:card', 'wire:click="deductCredits"', 'wire:click="refundCredits"'],
    );
});

it('renders balance and transaction history for the selected agent', function (): void {
    $component = $this->livewireComponent('CreditLedger');
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
    $component = $this->livewireComponent('CreditLedger');
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
