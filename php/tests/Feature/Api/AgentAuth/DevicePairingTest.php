<?php

// SPDX-License-Identifier: EUPL-1.2

declare(strict_types=1);

use Core\Mod\Agentic\Actions\Auth\CreateDevicePairing;
use Core\Mod\Agentic\Models\AgentApiKey;
use Core\Mod\Agentic\Models\DevicePairing;
use Core\Tenant\Models\Workspace;

beforeEach(function (): void {
    require __DIR__.'/../../../../Routes/api.php';
});

function devicePairing(Workspace $workspace, array $overrides = []): DevicePairing
{
    return CreateDevicePairing::run(
        $workspace->id,
        $overrides['user_id'] ?? null,
        $overrides['label'] ?? 'codex-local',
        $overrides['permissions'] ?? null,
        $overrides['rate_limit'] ?? 100,
        $overrides['key_ttl_days'] ?? null,
    );
}

test('login exchanges a valid pairing code for a new plaintext key', function (): void {
    $workspace = createWorkspace();
    $pairing = devicePairing($workspace);

    $response = $this->postJson('/v1/agent/auth/login', ['code' => $pairing->code]);

    $response
        ->assertOk()
        ->assertJsonPath('data.key.workspace_id', $workspace->id)
        ->assertJsonPath('data.key.name', 'codex-local')
        ->assertJsonPath('data.key.permissions.0', AgentApiKey::PERM_FLEET_READ);

    expect((string) $response->json('data.key.key'))->toStartWith('ak_');

    // The pairing is spent and the minted key is real.
    $pairing->refresh();
    expect($pairing->consumed_at)->not->toBeNull()
        ->and($pairing->agent_api_key_id)->toBe((int) $response->json('data.key.id'));
    expect(AgentApiKey::findByKey((string) $response->json('data.key.key')))->not->toBeNull();
});

test('login rejects an unknown code', function (): void {
    createWorkspace();

    $this->postJson('/v1/agent/auth/login', ['code' => '000000'])
        ->assertStatus(422)
        ->assertJsonPath('error', 'invalid_pairing_code');
});

test('a pairing code is single use', function (): void {
    $workspace = createWorkspace();
    $pairing = devicePairing($workspace);

    $this->postJson('/v1/agent/auth/login', ['code' => $pairing->code])->assertOk();
    $this->postJson('/v1/agent/auth/login', ['code' => $pairing->code])->assertStatus(422);
});

test('login rejects an expired code', function (): void {
    $workspace = createWorkspace();
    $pairing = devicePairing($workspace);
    $pairing->forceFill(['expires_at' => now()->subMinute()])->save();

    $this->postJson('/v1/agent/auth/login', ['code' => $pairing->code])
        ->assertStatus(422);
});

test('login validates the code shape', function (): void {
    $this->postJson('/v1/agent/auth/login', ['code' => 'abc'])
        ->assertStatus(422)
        ->assertJsonValidationErrorFor('code');
});

test('device pair returns the fleet-shaped api key payload', function (): void {
    $workspace = createWorkspace();
    $pairing = devicePairing($workspace, ['label' => 'fleet-node-1']);

    $response = $this->postJson('/v1/device/pair', ['code' => $pairing->code]);

    $response
        ->assertOk()
        ->assertJsonPath('data.agent_id', 'fleet-node-1');

    expect((string) $response->json('data.agent_api_key'))->toStartWith('ak_');
    expect($pairing->fresh()->consumed_at)->not->toBeNull();
});
