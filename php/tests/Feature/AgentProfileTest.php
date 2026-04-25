<?php

// SPDX-License-Identifier: EUPL-1.2

declare(strict_types=1);

use Core\Mod\Agentic\Models\AgentProfile;
use Illuminate\Foundation\Testing\RefreshDatabase;
use Illuminate\Support\Facades\DB;

uses(RefreshDatabase::class);

beforeEach(function (): void {
    if (! is_string(config('app.key')) || config('app.key') === '') {
        config(['app.key' => 'base64:'.base64_encode(random_bytes(32))]);
    }
});

it('returns only enabled profiles with remaining headroom from the active scope', function () {
    $activeProfile = AgentProfile::create([
        'name' => 'active-profile',
        'gateway_url' => 'https://gateway-a.example.com',
        'api_key_cipher' => 'secret-active',
        'cost_class' => 'A',
        'capability_tags' => ['dispatch'],
        'quota_headroom_pct' => 100,
        'enabled' => true,
    ]);

    AgentProfile::create([
        'name' => 'disabled-profile',
        'gateway_url' => 'https://gateway-b.example.com',
        'api_key_cipher' => 'secret-disabled',
        'cost_class' => 'A',
        'capability_tags' => ['dispatch'],
        'quota_headroom_pct' => 100,
        'enabled' => false,
    ]);

    AgentProfile::create([
        'name' => 'exhausted-profile',
        'gateway_url' => 'https://gateway-c.example.com',
        'api_key_cipher' => 'secret-exhausted',
        'cost_class' => 'A',
        'capability_tags' => ['dispatch'],
        'quota_headroom_pct' => 0,
        'enabled' => true,
    ]);

    $profiles = AgentProfile::active()->get();

    expect($profiles)->toHaveCount(1)
        ->and($profiles->first()->id)->toBe($activeProfile->id);
});

it('chains the active and forClass scopes correctly', function () {
    $matchingProfile = AgentProfile::create([
        'name' => 'class-a-active',
        'gateway_url' => 'https://gateway-d.example.com',
        'api_key_cipher' => 'secret-class-a',
        'cost_class' => 'A',
        'capability_tags' => ['dispatch'],
        'quota_headroom_pct' => 75,
        'enabled' => true,
    ]);

    AgentProfile::create([
        'name' => 'class-b-active',
        'gateway_url' => 'https://gateway-e.example.com',
        'api_key_cipher' => 'secret-class-b',
        'cost_class' => 'B',
        'capability_tags' => ['dispatch'],
        'quota_headroom_pct' => 75,
        'enabled' => true,
    ]);

    AgentProfile::create([
        'name' => 'class-a-disabled',
        'gateway_url' => 'https://gateway-f.example.com',
        'api_key_cipher' => 'secret-class-a-disabled',
        'cost_class' => 'A',
        'capability_tags' => ['dispatch'],
        'quota_headroom_pct' => 75,
        'enabled' => false,
    ]);

    AgentProfile::create([
        'name' => 'class-a-exhausted',
        'gateway_url' => 'https://gateway-g.example.com',
        'api_key_cipher' => 'secret-class-a-exhausted',
        'cost_class' => 'A',
        'capability_tags' => ['dispatch'],
        'quota_headroom_pct' => 0,
        'enabled' => true,
    ]);

    $profiles = AgentProfile::active()->forClass('A')->get();

    expect($profiles)->toHaveCount(1)
        ->and($profiles->first()->id)->toBe($matchingProfile->id);
});

it('round-trips the api_key_cipher attribute through the encrypted cast', function () {
    $profile = AgentProfile::create([
        'name' => 'encrypted-profile',
        'gateway_url' => 'https://gateway-h.example.com',
        'api_key_cipher' => 'plain-secret-value',
        'cost_class' => 'A',
        'capability_tags' => ['dispatch', 'review'],
        'quota_headroom_pct' => 100,
        'enabled' => true,
    ]);

    $rawCiphertext = DB::table('agent_profiles')
        ->where('id', $profile->id)
        ->value('api_key_cipher');

    expect($rawCiphertext)->not->toBe('plain-secret-value')
        ->and($profile->fresh()->api_key_cipher)->toBe('plain-secret-value');
});

it('round-trips capability tags as an array', function () {
    $profile = AgentProfile::create([
        'name' => 'tagged-profile',
        'gateway_url' => 'https://gateway-i.example.com',
        'api_key_cipher' => 'plain-secret-tags',
        'cost_class' => 'C',
        'capability_tags' => ['dispatch', 'analysis', 'handoff'],
        'quota_headroom_pct' => 100,
        'enabled' => true,
    ]);

    expect($profile->fresh()->capability_tags)->toBe([
        'dispatch',
        'analysis',
        'handoff',
    ]);
});
