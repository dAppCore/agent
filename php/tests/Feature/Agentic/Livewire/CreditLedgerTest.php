<?php

// SPDX-License-Identifier: EUPL-1.2

declare(strict_types=1);

namespace Core\Mod\Agentic\Tests\Feature\Agentic\Livewire;

use Core\Mod\Agentic\Actions\Credits\AwardCredits;
use Core\Mod\Agentic\Actions\Credits\GetBalance;
use Core\Mod\Agentic\Models\FleetNode;
use Core\Mod\Agentic\Tests\Feature\Livewire\LivewireTestCase;
use Livewire\Livewire;

/**
 * Class-style for the same reason as BrainExplorerTest: a file-level
 * uses(LivewireTestCase::class) collides with the directory-wide
 * uses(Tests\TestCase::class) that Pest.php applies to Feature.
 */
class CreditLedgerTest extends LivewireTestCase
{
    protected function setUp(): void
    {
        parent::setUp();

        $this->actingAsHades();
    }

    public function test_wires_credit_actions_and_flux_blade_controls(): void
    {
        $this->assertFluxComponentWiring(
            'CreditLedger',
            'credit-ledger',
            ['AwardCredits', 'GetBalance', 'GetCreditHistory'],
            ['<flux:card', 'wire:click="deductCredits"', 'wire:click="refundCredits"'],
        );
    }

    public function test_renders_balance_and_transaction_history_for_the_selected_agent(): void
    {
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
    }

    public function test_refunds_and_deducts_credits_through_the_ledger_actions(): void
    {
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

        $this->assertDatabaseHas('credit_entries', [
            'workspace_id' => $workspace->id,
            'task_type' => 'manual-refund',
            'amount' => 3,
            'description' => 'Manual refund',
        ]);

        $this->assertDatabaseHas('credit_entries', [
            'workspace_id' => $workspace->id,
            'task_type' => 'manual-deduction',
            'amount' => -2,
            'description' => 'Manual deduction',
        ]);

        $this->assertSame(1, GetBalance::run($workspace->id, 'alpha')['balance']);
    }
}
