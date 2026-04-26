<?php

// SPDX-License-Identifier: EUPL-1.2

declare(strict_types=1);

use Core\Mod\Agentic\Console\Commands\AgenticSyncProfilesCommand;
use Core\Mod\Agentic\Models\AgentProfile;
use Illuminate\Console\Command;
use Illuminate\Contracts\Console\Kernel;
use Illuminate\Http\Client\Request;
use Illuminate\Support\Carbon;
use Illuminate\Support\Facades\Http;

beforeEach(function (): void {
    if (! is_string(config('app.key')) || config('app.key') === '') {
        config(['app.key' => 'base64:'.base64_encode(random_bytes(32))]);
    }

    $this->app->make(Kernel::class)->registerCommand(
        $this->app->make(AgenticSyncProfilesCommand::class),
    );
});

function syncProfilesProfile(array $attributes = []): AgentProfile
{
    return AgentProfile::create(array_merge([
        'name' => $attributes['name'] ?? 'dispatch-profile',
        'gateway_url' => $attributes['gateway_url'] ?? 'https://gateway.example.test',
        'api_key_cipher' => $attributes['api_key_cipher'] ?? 'plain-secret',
        'cost_class' => 'C',
        'capability_tags' => ['dispatch'],
        'quota_headroom_pct' => 100,
        'enabled' => true,
        'last_dispatched_at' => null,
    ], $attributes));
}

test('AgenticSyncProfilesCommand_handle_Good_synchronises_gateway_models_without_touching_dispatch_usage', function (): void {
    $lastDispatchedAt = Carbon::parse('2026-04-26 09:15:00');

    $opusProfile = syncProfilesProfile([
        'name' => 'claude-opus-profile',
        'gateway_url' => 'https://gateway-opus.example.test',
        'api_key_cipher' => 'opus-token',
        'capability_tags' => ['stale'],
        'last_dispatched_at' => $lastDispatchedAt,
    ]);

    $miniProfile = syncProfilesProfile([
        'name' => 'gpt-mini-profile',
        'gateway_url' => 'https://gateway-mini.example.test',
        'api_key_cipher' => 'mini-token',
        'capability_tags' => ['old-tag'],
        'enabled' => false,
    ]);

    Http::fake([
        'https://gateway-opus.example.test/v1/models' => Http::response([
            'data' => [
                ['id' => 'claude-opus-4-1'],
                ['id' => 'embedding-3-large'],
            ],
        ], 200),
        'https://gateway-mini.example.test/v1/models' => Http::response([
            'data' => [
                ['id' => 'gpt-5.4-mini'],
            ],
        ], 200),
    ]);

    $this->artisan('agentic:sync-profiles')
        ->expectsOutputToContain('Synchronised 2 profile(s); 0 failed.')
        ->assertSuccessful();

    Http::assertSent(fn (Request $request): bool => $request->method() === 'GET'
        && $request->url() === 'https://gateway-opus.example.test/v1/models'
        && $request->hasHeader('Authorization', 'Bearer opus-token'));

    Http::assertSent(fn (Request $request): bool => $request->method() === 'GET'
        && $request->url() === 'https://gateway-mini.example.test/v1/models'
        && $request->hasHeader('Authorization', 'Bearer mini-token'));

    expect($opusProfile->fresh()->capability_tags)->toBe([
        'analysis',
        'core',
        'embedding',
        'handoff',
    ])->and($opusProfile->fresh()->last_dispatched_at?->equalTo($lastDispatchedAt))->toBeTrue()
        ->and($miniProfile->fresh()->capability_tags)->toBe([
            'cheap',
            'dispatch',
        ]);
});

test('AgenticSyncProfilesCommand_handle_Bad_continues_past_gateway_failures_and_returns_failure', function (): void {
    $healthyProfile = syncProfilesProfile([
        'name' => 'healthy-profile',
        'gateway_url' => 'https://gateway-healthy.example.test',
        'api_key_cipher' => 'healthy-token',
    ]);

    $brokenProfile = syncProfilesProfile([
        'name' => 'broken-profile',
        'gateway_url' => 'https://gateway-broken.example.test',
        'api_key_cipher' => 'broken-token',
        'capability_tags' => ['keep-me'],
    ]);

    Http::fake([
        'https://gateway-healthy.example.test/v1/models' => Http::response([
            'data' => [
                ['id' => 'gpt-5.4-mini'],
            ],
        ], 200),
        'https://gateway-broken.example.test/v1/models' => Http::response([], 500),
    ]);

    $this->artisan('agentic:sync-profiles')
        ->expectsOutputToContain('Failed to synchronise broken-profile: Gateway models request failed: 500')
        ->expectsOutputToContain('Synchronised 1 profile(s); 1 failed.')
        ->assertExitCode(Command::FAILURE);

    expect($healthyProfile->fresh()->capability_tags)->toBe([
        'cheap',
        'dispatch',
    ])->and($brokenProfile->fresh()->capability_tags)->toBe([
        'keep-me',
    ]);
});
