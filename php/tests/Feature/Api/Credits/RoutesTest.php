<?php

// SPDX-License-Identifier: EUPL-1.2

declare(strict_types=1);

use Core\Mod\Agentic\Models\AgentApiKey;
use Core\Mod\Agentic\Models\CreditEntry;
use Core\Tenant\Models\Workspace;

beforeEach(function (): void {
    require __DIR__.'/../../../../Routes/api.php';
});

function creditsRouteKey(
    Workspace $workspace,
    array $permissions = [AgentApiKey::PERM_CREDITS_READ, AgentApiKey::PERM_CREDITS_WRITE]
): AgentApiKey {
    return createApiKey($workspace, 'Credits Route Key', $permissions);
}

test('credits balance route returns workspace totals', function (): void {
    $workspace = createWorkspace();
    $key = creditsRouteKey($workspace, [AgentApiKey::PERM_CREDITS_READ]);

    CreditEntry::create([
        'workspace_id' => $workspace->id,
        'fleet_node_id' => null,
        'task_type' => 'manual-refund',
        'amount' => 30,
        'balance_after' => 30,
    ]);
    CreditEntry::create([
        'workspace_id' => $workspace->id,
        'fleet_node_id' => null,
        'task_type' => 'manual-deduction',
        'amount' => -10,
        'balance_after' => 20,
    ]);

    $response = $this
        ->withHeader('Authorization', 'Bearer '.$key->plainTextKey)
        ->getJson('/v1/credits/balance');

    $response
        ->assertOk()
        ->assertJsonPath('data.workspace_id', $workspace->id)
        ->assertJsonPath('data.balance', 20)
        ->assertJsonPath('data.total_earned', 30)
        ->assertJsonPath('data.total_spent', 10)
        ->assertJsonPath('data.entries', 2);
});

test('credits deduct route records a negative ledger entry', function (): void {
    $workspace = createWorkspace();
    $key = creditsRouteKey($workspace, [AgentApiKey::PERM_CREDITS_WRITE]);

    $response = $this
        ->withHeader('Authorization', 'Bearer '.$key->plainTextKey)
        ->postJson('/v1/credits/deduct', [
            'amount' => 15,
            'reason' => 'Manual moderation charge',
        ]);

    $response
        ->assertCreated()
        ->assertJsonPath('data.amount', -15)
        ->assertJsonPath('data.balance_after', -15)
        ->assertJsonPath('data.task_type', 'manual-deduction');
});

test('credits refund route records a positive ledger entry', function (): void {
    $workspace = createWorkspace();
    $key = creditsRouteKey($workspace, [AgentApiKey::PERM_CREDITS_WRITE]);

    $response = $this
        ->withHeader('Authorization', 'Bearer '.$key->plainTextKey)
        ->postJson('/v1/credits/refund', [
            'amount' => 25,
            'reason' => 'Manual goodwill refund',
        ]);

    $response
        ->assertCreated()
        ->assertJsonPath('data.amount', 25)
        ->assertJsonPath('data.balance_after', 25)
        ->assertJsonPath('data.task_type', 'manual-refund');
});

test('credits ledger route returns the newest workspace entries first', function (): void {
    $workspace = createWorkspace();
    $key = creditsRouteKey($workspace, [AgentApiKey::PERM_CREDITS_READ]);

    CreditEntry::create([
        'workspace_id' => $workspace->id,
        'fleet_node_id' => null,
        'task_type' => 'manual-refund',
        'amount' => 10,
        'balance_after' => 10,
    ]);
    CreditEntry::create([
        'workspace_id' => $workspace->id,
        'fleet_node_id' => null,
        'task_type' => 'manual-deduction',
        'amount' => -3,
        'balance_after' => 7,
    ]);

    $response = $this
        ->withHeader('Authorization', 'Bearer '.$key->plainTextKey)
        ->getJson('/v1/credits/ledger');

    $response
        ->assertOk()
        ->assertJsonPath('total', 2)
        ->assertJsonPath('data.0.task_type', 'manual-deduction')
        ->assertJsonPath('data.0.balance_after', 7)
        ->assertJsonPath('data.1.task_type', 'manual-refund');
});
