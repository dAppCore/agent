<?php

// SPDX-License-Identifier: EUPL-1.2

declare(strict_types=1);

use Core\Mod\Agentic\Models\AgentApiKey;
use Core\Mod\Agentic\Models\CreditEntry;
use Core\Mod\Agentic\Models\FleetNode;
use Core\Tenant\Models\Workspace;

beforeEach(function (): void {
    require __DIR__.'/../../../../Routes/api.php';
});

function subscriptionRouteKey(
    Workspace $workspace,
    array $permissions = [AgentApiKey::PERM_SUBSCRIPTION_READ, AgentApiKey::PERM_SUBSCRIPTION_WRITE]
): AgentApiKey {
    return createApiKey($workspace, 'Subscription Route Key', $permissions);
}

test('subscription status route reports capability and credit status', function (): void {
    $workspace = createWorkspace();
    $key = subscriptionRouteKey($workspace, [AgentApiKey::PERM_SUBSCRIPTION_READ]);

    CreditEntry::create([
        'workspace_id' => $workspace->id,
        'fleet_node_id' => null,
        'task_type' => 'manual-refund',
        'amount' => 5,
        'balance_after' => 5,
    ]);

    $response = $this
        ->withHeader('Authorization', 'Bearer '.$key->plainTextKey)
        ->getJson('/v1/subscription/status?api_keys[openai]=test-key');

    $response
        ->assertOk()
        ->assertJsonPath('data.status', 'active')
        ->assertJsonPath('data.providers.openai', true)
        ->assertJsonPath('data.credits.balance', 5);
});

test('subscription upgrade route updates the node budget', function (): void {
    $workspace = createWorkspace();
    $key = subscriptionRouteKey($workspace, [AgentApiKey::PERM_SUBSCRIPTION_WRITE]);

    FleetNode::create([
        'workspace_id' => $workspace->id,
        'agent_id' => 'charon',
        'platform' => 'linux',
        'status' => FleetNode::STATUS_ONLINE,
        'compute_budget' => ['max_daily_hours' => 1],
    ]);

    $response = $this
        ->withHeader('Authorization', 'Bearer '.$key->plainTextKey)
        ->postJson('/v1/subscription/upgrade', [
            'agent_id' => 'charon',
            'limits' => ['max_daily_hours' => 4, 'prefer_models' => ['codex:gpt-5.4-mini']],
        ]);

    $response
        ->assertOk()
        ->assertJsonPath('data.status', 'upgraded')
        ->assertJsonPath('data.budget.max_daily_hours', 4)
        ->assertJsonPath('data.budget.prefer_models.0', 'codex:gpt-5.4-mini');
});

test('subscription cancel route marks the node budget as cancelled', function (): void {
    $workspace = createWorkspace();
    $key = subscriptionRouteKey($workspace, [AgentApiKey::PERM_SUBSCRIPTION_WRITE]);

    FleetNode::create([
        'workspace_id' => $workspace->id,
        'agent_id' => 'virgil',
        'platform' => 'linux',
        'status' => FleetNode::STATUS_ONLINE,
        'compute_budget' => ['max_daily_hours' => 8],
    ]);

    $response = $this
        ->withHeader('Authorization', 'Bearer '.$key->plainTextKey)
        ->postJson('/v1/subscription/cancel', [
            'agent_id' => 'virgil',
        ]);

    $response
        ->assertOk()
        ->assertJsonPath('data.status', 'cancelled')
        ->assertJsonPath('data.budget.cancelled', true)
        ->assertJsonPath('data.budget.max_daily_hours', 0);
});
