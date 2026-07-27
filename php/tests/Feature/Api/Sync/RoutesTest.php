<?php

// SPDX-License-Identifier: EUPL-1.2

declare(strict_types=1);

// NOTE: updated for the fleet reconciliation — the sync actions resolve agent
// identity via AgentRegistration (sync_records re-keyed to workspace_id +
// agent_id). Flagged UNRUN: framework test suite can't be installed here (forge
// offline); verify in CI.

use Core\Mod\Agentic\Models\AgentApiKey;
use Core\Mod\Agentic\Models\AgentRegistration;
use Core\Mod\Agentic\Models\BrainMemory;
use Core\Tenant\Models\Workspace;

beforeEach(function (): void {
    require __DIR__.'/../../../../Routes/api.php';
});

function syncRouteKey(
    Workspace $workspace,
    array $permissions = [AgentApiKey::PERM_SYNC_READ, AgentApiKey::PERM_SYNC_WRITE]
): AgentApiKey {
    return createApiKey($workspace, 'Sync Route Key', $permissions);
}

test('agent sync push route stores dispatch history', function (): void {
    $workspace = createWorkspace();
    $key = syncRouteKey($workspace, [AgentApiKey::PERM_SYNC_WRITE]);

    $response = $this
        ->withHeader('Authorization', 'Bearer '.$key->plainTextKey)
        ->postJson('/v1/agent/sync/push', [
            'agent_id' => 'charon',
            'dispatches' => [[
                'repo' => 'dappco.re/go/agent',
                'workspace' => 'core-agent',
                'task' => 'Record the sync alias route',
                'status' => 'completed',
            ]],
        ]);

    $response
        ->assertCreated()
        ->assertJsonPath('data.synced', 1);

    expect(
        AgentRegistration::query()
            ->where('workspace_id', $workspace->id)
            ->where('agent_id', 'charon')
            ->exists()
    )->toBeTrue();
});

test('agent sync pull route returns shared context', function (): void {
    $workspace = createWorkspace();
    $key = syncRouteKey($workspace, [AgentApiKey::PERM_SYNC_READ]);

    AgentRegistration::create([
        'workspace_id' => $workspace->id,
        'agent_id' => 'charon',
        'hostname' => 'charon',
        'platform' => 'linux',
        'status' => AgentRegistration::STATUS_ONLINE,
    ]);

    BrainMemory::create([
        'workspace_id' => $workspace->id,
        'agent_id' => 'charon',
        'type' => 'observation',
        'content' => 'Shared context for the new pull route.',
        'tags' => ['sync'],
        'confidence' => 0.8,
        'source' => 'test',
    ]);

    $response = $this
        ->withHeader('Authorization', 'Bearer '.$key->plainTextKey)
        ->getJson('/v1/agent/sync/pull?agent_id=charon');

    $response
        ->assertOk()
        ->assertJsonPath('total', 1)
        ->assertJsonPath('data.0.agent_id', 'charon')
        ->assertJsonPath('data.0.content', 'Shared context for the new pull route.');
});
