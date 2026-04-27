<?php

// SPDX-License-Identifier: EUPL-1.2

declare(strict_types=1);

use Core\Mod\Agentic\Models\AgentApiKey;
use Core\Mod\Agentic\Models\AgentSession;
use Core\Tenant\Models\Workspace;

beforeEach(function (): void {
    require __DIR__.'/../../../../Routes/api.php';
});

function agentAuthRouteKey(
    Workspace $workspace,
    array $permissions = [AgentApiKey::PERM_AUTH_WRITE, AgentApiKey::PERM_SESSIONS_WRITE]
): AgentApiKey {
    return createApiKey($workspace, 'Agent Auth Key', $permissions);
}

test('agent auth register route creates an authenticated session record', function (): void {
    $workspace = createWorkspace();
    $key = agentAuthRouteKey($workspace);

    $response = $this
        ->withHeader('Authorization', 'Bearer '.$key->plainTextKey)
        ->postJson('/v1/agent/auth/register', [
            'agent_type' => 'codex',
            'context' => ['repo' => 'core/agent'],
            'work_log' => [
                ['message' => 'Registered via API'],
            ],
        ]);

    $response
        ->assertCreated()
        ->assertJsonPath('data.status', AgentSession::STATUS_ACTIVE)
        ->assertJsonPath('data.agent_type', 'codex')
        ->assertJsonPath('data.context_summary.repo', 'core/agent')
        ->assertJsonPath('data.work_log.0.message', 'Registered via API');

    expect(AgentSession::query()->where('workspace_id', $workspace->id)->count())->toBe(1);
});

test('agent auth provision route returns a new plain text key', function (): void {
    $workspace = createWorkspace();

    $this->withoutMiddleware();

    $response = $this->postJson('/v1/agent/auth/provision', [
        'workspace_id' => $workspace->id,
        'oauth_user_id' => 'user-42',
        'permissions' => [AgentApiKey::PERM_FLEET_READ],
        'rate_limit' => 120,
    ]);

    $response
        ->assertCreated()
        ->assertJsonPath('data.rate_limit', 120)
        ->assertJsonPath('data.permissions.0', AgentApiKey::PERM_FLEET_READ);

    expect((string) $response->json('data.plain_text_key'))->toStartWith('ak_');
});
