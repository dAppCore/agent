<?php

// SPDX-License-Identifier: EUPL-1.2

declare(strict_types=1);

use Core\Mod\Agentic\Models\AgentProfile;
use Core\Mod\Agentic\Services\ProfileSelector;
use Illuminate\Foundation\Testing\RefreshDatabase;
use Illuminate\Support\Carbon;

uses(RefreshDatabase::class);

beforeEach(function (): void {
    if (! is_string(config('app.key')) || config('app.key') === '') {
        config(['app.key' => 'base64:'.base64_encode(random_bytes(32))]);
    }

    Carbon::setTestNow('2026-04-26 12:00:00');
});

afterEach(function (): void {
    Carbon::setTestNow();
});

function seedAgentProfile(array $overrides = []): AgentProfile
{
    static $sequence = 0;

    $sequence++;

    return AgentProfile::create(array_merge([
        'name' => "profile-{$sequence}",
        'gateway_url' => "https://gateway-{$sequence}.example.com",
        'api_key_cipher' => "secret-{$sequence}",
        'cost_class' => 'C',
        'capability_tags' => ['dispatch'],
        'quota_headroom_pct' => 100,
        'enabled' => true,
        'last_dispatched_at' => null,
    ], $overrides));
}

test('ProfileSelector_pickFor_Good_picks_the_cheapest_class_A_profile_with_matching_capability', function (): void {
    $selector = new ProfileSelector;

    $cheapestMatch = seedAgentProfile([
        'name' => 'security-a',
        'cost_class' => 'A',
        'capability_tags' => ['security'],
    ]);

    seedAgentProfile([
        'name' => 'security-b',
        'cost_class' => 'B',
        'capability_tags' => ['security'],
    ]);

    seedAgentProfile([
        'name' => 'dispatch-a',
        'cost_class' => 'A',
        'capability_tags' => ['dispatch'],
    ]);

    $picked = $selector->pickFor([
        'id' => 826,
        'severity' => 'critical',
        'priority' => 'normal',
        'project' => 'mantis',
        'category' => 'security',
        'summary' => 'Triage a critical security issue',
        'tags' => ['security'],
    ]);

    expect($picked?->id)->toBe($cheapestMatch->id)
        ->and($picked?->last_dispatched_at)->not->toBeNull();
});

test('ProfileSelector_pickFor_Good_requires_headroom_for_class_B_matches', function (): void {
    $selector = new ProfileSelector;

    seedAgentProfile([
        'name' => 'review-low-headroom',
        'capability_tags' => ['review'],
        'quota_headroom_pct' => 20,
    ]);

    $eligible = seedAgentProfile([
        'name' => 'review-eligible',
        'capability_tags' => ['review'],
        'quota_headroom_pct' => 25,
    ]);

    seedAgentProfile([
        'name' => 'review-disabled',
        'capability_tags' => ['review'],
        'quota_headroom_pct' => 90,
        'enabled' => false,
    ]);

    $picked = $selector->pickFor([
        'id' => 827,
        'severity' => 'major',
        'priority' => 'normal',
        'project' => 'mantis',
        'category' => 'review',
        'summary' => 'High-impact review task',
        'tags' => ['review'],
    ]);

    expect($picked?->id)->toBe($eligible->id);
});

test('ProfileSelector_pickFor_Good_round_robins_class_C_profiles_deterministically', function (): void {
    $selector = new ProfileSelector;

    $first = seedAgentProfile(['name' => 'round-robin-1']);
    $second = seedAgentProfile(['name' => 'round-robin-2']);
    $third = seedAgentProfile(['name' => 'round-robin-3']);

    $ticket = [
        'id' => 828,
        'severity' => 'minor',
        'priority' => 'normal',
        'project' => 'mantis',
        'category' => 'general',
        'summary' => 'General maintenance',
        'tags' => [],
    ];

    $pickedIds = [
        $selector->pickFor($ticket)?->id,
        $selector->pickFor($ticket)?->id,
        $selector->pickFor($ticket)?->id,
        $selector->pickFor($ticket)?->id,
    ];

    expect($pickedIds)->toBe([
        $first->id,
        $second->id,
        $third->id,
        $first->id,
    ]);
});

test('ProfileSelector_pickFor_Bad_returns_null_when_no_matching_capability_exists', function (): void {
    $selector = new ProfileSelector;

    seedAgentProfile([
        'name' => 'dispatch-only',
        'cost_class' => 'A',
        'capability_tags' => ['dispatch'],
    ]);

    $picked = $selector->pickFor([
        'id' => 829,
        'severity' => 'critical',
        'priority' => 'normal',
        'project' => 'mantis',
        'category' => 'crypto',
        'summary' => 'Needs crypto specialist',
        'tags' => ['crypto'],
    ]);

    expect($picked)->toBeNull();
});

test('ProfileSelector_pickFor_Ugly_returns_null_when_all_profiles_are_disabled', function (): void {
    $selector = new ProfileSelector;

    seedAgentProfile([
        'name' => 'disabled-one',
        'enabled' => false,
        'capability_tags' => ['dispatch'],
    ]);

    seedAgentProfile([
        'name' => 'disabled-two',
        'enabled' => false,
        'capability_tags' => ['review'],
    ]);

    $picked = $selector->pickFor([
        'id' => 830,
        'severity' => 'minor',
        'priority' => 'normal',
        'project' => 'mantis',
        'category' => 'general',
        'summary' => 'No active profiles left',
        'tags' => [],
    ]);

    expect($picked)->toBeNull();
});
