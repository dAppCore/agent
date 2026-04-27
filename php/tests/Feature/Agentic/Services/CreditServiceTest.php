<?php

// SPDX-License-Identifier: EUPL-1.2

declare(strict_types=1);

use Core\Mod\Agentic\Data\CreditTransaction;
use Core\Mod\Agentic\Models\CreditEntry;
use Core\Mod\Agentic\Models\FleetNode;
use Core\Mod\Agentic\Services\CreditService;

use function Pest\Laravel\assertDatabaseHas;

if (! function_exists('loadAgenticPhpClass')) {
    function loadAgenticPhpClass(string $relativePath): void
    {
        $phpRoot = dirname(__DIR__, 4);
        require_once $phpRoot.'/'.$relativePath;
    }
}

beforeEach(function (): void {
    loadAgenticPhpClass('Agentic/Data/CreditTransaction.php');
    loadAgenticPhpClass('Agentic/Services/CreditService.php');
});

test('CreditService_refundAndDeduct_Good_tracks_workspace_balance_and_ledger_entries', function (): void {
    $workspace = createWorkspace();
    $service = new CreditService();

    $refund = $service->refund($workspace->id, 7, 'Initial workspace credit');
    $deduction = $service->deduct($workspace->id, 2, 'Dispatch overrun');
    $balance = $service->balance($workspace->id);
    $ledger = $service->ledger($workspace->id);

    expect($refund)->toBeInstanceOf(CreditTransaction::class)
        ->and($deduction)->toBeInstanceOf(CreditTransaction::class)
        ->and($balance['balance'])->toBe(5)
        ->and($balance['total_earned'])->toBe(7)
        ->and($balance['total_spent'])->toBe(2)
        ->and($ledger)->toHaveCount(2)
        ->and($ledger->first()->taskType)->toBe('manual-deduction');

    assertDatabaseHas('credit_entries', [
        'workspace_id' => $workspace->id,
        'fleet_node_id' => null,
        'task_type' => 'manual-refund',
        'amount' => 7,
        'balance_after' => 7,
    ]);

    assertDatabaseHas('credit_entries', [
        'workspace_id' => $workspace->id,
        'fleet_node_id' => null,
        'task_type' => 'manual-deduction',
        'amount' => -2,
        'balance_after' => 5,
    ]);
});

test('CreditService_deduct_Bad_rejects_zero_amounts_and_blank_reasons', function (): void {
    $workspace = createWorkspace();
    $service = new CreditService();

    expect(fn () => $service->deduct($workspace->id, 0, 'No-op'))
        ->toThrow(InvalidArgumentException::class, 'amount must be greater than zero');

    expect(fn () => $service->refund($workspace->id, 5, '   '))
        ->toThrow(InvalidArgumentException::class, 'reason is required');
});

test('CreditService_balance_Ugly_aggregates_existing_node_entries_with_workspace_level_adjustments', function (): void {
    $workspace = createWorkspace();
    $node = FleetNode::query()->create([
        'workspace_id' => $workspace->id,
        'agent_id' => 'gamma',
        'platform' => 'linux',
        'status' => FleetNode::STATUS_ONLINE,
        'registered_at' => now(),
        'last_heartbeat_at' => now(),
    ]);

    CreditEntry::query()->create([
        'workspace_id' => $workspace->id,
        'fleet_node_id' => $node->id,
        'task_type' => 'task-complete',
        'amount' => 6,
        'balance_after' => 6,
        'description' => 'Legacy per-node award',
    ]);

    $service = new CreditService();
    $service->refund($workspace->id, 4, 'Workspace top-up');

    $balance = $service->balance($workspace->id);
    $ledger = $service->ledger($workspace->id);

    expect($balance['balance'])->toBe(10)
        ->and($balance['entries'])->toBe(2)
        ->and($ledger->map(fn (CreditTransaction $entry): string => $entry->taskType)->all())
        ->toBe(['manual-refund', 'task-complete']);
});
