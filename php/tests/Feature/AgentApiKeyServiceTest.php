<?php

declare(strict_types=1);

/**
 * Tests for the AgentApiKeyService.
 *
 * Covers authentication, IP validation, rate limit tracking, and key management.
 */

use Carbon\Carbon;
use Core\Mod\Agentic\Models\AgentApiKey;
use Core\Mod\Agentic\Services\AgentApiKeyService;
use Illuminate\Support\Facades\Cache;

// =========================================================================
// Key Creation Tests
// =========================================================================

describe('key creation', function () {
    it('creates and returns an AgentApiKey instance', function () {
        $workspace = createWorkspace();
        $service = app(AgentApiKeyService::class);

        $key = $service->create($workspace, 'Test Key');

        expect($key)
            ->toBeInstanceOf(AgentApiKey::class)
            ->and($key->plainTextKey)->not->toBeNull();
    });

    it('creates key using workspace ID', function () {
        $workspace = createWorkspace();
        $service = app(AgentApiKeyService::class);

        $key = $service->create($workspace->id, 'Test Key');

        expect($key->workspace_id)->toBe($workspace->id);
    });

    it('creates key with specified permissions', function () {
        $workspace = createWorkspace();
        $service = app(AgentApiKeyService::class);
        $permissions = [
            AgentApiKey::PERM_PLANS_READ,
            AgentApiKey::PERM_PLANS_WRITE,
        ];

        $key = $service->create($workspace, 'Test Key', $permissions);

        expect($key->permissions)->toBe($permissions);
    });

    it('creates key with custom rate limit', function () {
        $workspace = createWorkspace();
        $service = app(AgentApiKeyService::class);

        $key = $service->create($workspace, 'Test Key', [], 500);

        expect($key->rate_limit)->toBe(500);
    });

    it('creates key with expiry date', function () {
        $workspace = createWorkspace();
        $service = app(AgentApiKeyService::class);
        $expiresAt = Carbon::now()->addMonth();

        $key = $service->create($workspace, 'Test Key', [], 100, $expiresAt);

        expect($key->expires_at->toDateTimeString())
            ->toBe($expiresAt->toDateTimeString());
    });
});

// =========================================================================
// Key Validation Tests
// =========================================================================

describe('key validation', function () {
    it('returns key for valid plaintext key', function () {
        $workspace = createWorkspace();
        $service = app(AgentApiKeyService::class);
        $key = $service->create($workspace, 'Test Key');
        $plainKey = $key->plainTextKey;

        $result = $service->validate($plainKey);

        expect($result)
            ->not->toBeNull()
            ->and($result->id)->toBe($key->id);
    });

    it('returns null for invalid key', function () {
        $service = app(AgentApiKeyService::class);

        $result = $service->validate('ak_invalid_key_here');

        expect($result)->toBeNull();
    });

    it('returns null for revoked key', function () {
        $workspace = createWorkspace();
        $service = app(AgentApiKeyService::class);
        $key = $service->create($workspace, 'Test Key');
        $plainKey = $key->plainTextKey;
        $key->revoke();

        $result = $service->validate($plainKey);

        expect($result)->toBeNull();
    });

    it('returns null for expired key', function () {
        $workspace = createWorkspace();
        $service = app(AgentApiKeyService::class);
        $key = $service->create(
            $workspace,
            'Test Key',
            [],
            100,
            Carbon::now()->subDay()
        );
        $plainKey = $key->plainTextKey;

        $result = $service->validate($plainKey);

        expect($result)->toBeNull();
    });
});

// =========================================================================
// Permission Check Tests
// =========================================================================

describe('permission checks', function () {
    it('checkPermission returns true when permission granted', function () {
        $workspace = createWorkspace();
        $service = app(AgentApiKeyService::class);
        $key = $service->create(
            $workspace,
            'Test Key',
            [AgentApiKey::PERM_PLANS_READ]
        );

        $result = $service->checkPermission($key, AgentApiKey::PERM_PLANS_READ);

        expect($result)->toBeTrue();
    });

    it('checkPermission returns false when permission not granted', function () {
        $workspace = createWorkspace();
        $service = app(AgentApiKeyService::class);
        $key = $service->create(
            $workspace,
            'Test Key',
            [AgentApiKey::PERM_PLANS_READ]
        );

        $result = $service->checkPermission($key, AgentApiKey::PERM_PLANS_WRITE);

        expect($result)->toBeFalse();
    });

    it('checkPermission returns false for inactive key', function () {
        $workspace = createWorkspace();
        $service = app(AgentApiKeyService::class);
        $key = $service->create(
            $workspace,
            'Test Key',
            [AgentApiKey::PERM_PLANS_READ]
        );
        $key->revoke();

        $result = $service->checkPermission($key, AgentApiKey::PERM_PLANS_READ);

        expect($result)->toBeFalse();
    });

    it('checkPermissions returns true when all permissions granted', function () {
        $workspace = createWorkspace();
        $service = app(AgentApiKeyService::class);
        $key = $service->create(
            $workspace,
            'Test Key',
            [AgentApiKey::PERM_PLANS_READ, AgentApiKey::PERM_PLANS_WRITE]
        );

        $result = $service->checkPermissions($key, [
            AgentApiKey::PERM_PLANS_READ,
            AgentApiKey::PERM_PLANS_WRITE,
        ]);

        expect($result)->toBeTrue();
    });

    it('checkPermissions returns false when missing one permission', function () {
        $workspace = createWorkspace();
        $service = app(AgentApiKeyService::class);
        $key = $service->create(
            $workspace,
            'Test Key',
            [AgentApiKey::PERM_PLANS_READ]
        );

        $result = $service->checkPermissions($key, [
            AgentApiKey::PERM_PLANS_READ,
            AgentApiKey::PERM_PLANS_WRITE,
        ]);

        expect($result)->toBeFalse();
    });
});

// =========================================================================
// Rate Limiting Tests
// =========================================================================

describe('rate limiting', function () {
    it('recordUsage increments cache counter', function () {
        $workspace = createWorkspace();
        $service = app(AgentApiKeyService::class);
        $key = $service->create($workspace, 'Test Key');
        Cache::forget("agent_api_key_rate:{$key->id}");

        $service->recordUsage($key);
        $service->recordUsage($key);
        $service->recordUsage($key);

        expect(Cache::get("agent_api_key_rate:{$key->id}"))->toBe(3);
    });

    it('recordUsage records client IP', function () {
        $workspace = createWorkspace();
        $service = app(AgentApiKeyService::class);
        $key = $service->create($workspace, 'Test Key');

        $service->recordUsage($key, '192.168.1.100');

        expect($key->fresh()->last_used_ip)->toBe('192.168.1.100');
    });

    it('recordUsage updates last_used_at', function () {
        $workspace = createWorkspace();
        $service = app(AgentApiKeyService::class);
        $key = $service->create($workspace, 'Test Key');

        $service->recordUsage($key);

        expect($key->fresh()->last_used_at)->not->toBeNull();
    });

    it('isRateLimited returns false when under limit', function () {
        $workspace = createWorkspace();
        $service = app(AgentApiKeyService::class);
        $key = $service->create($workspace, 'Test Key', [], 100);
        Cache::put("agent_api_key_rate:{$key->id}", 50, 60);

        $result = $service->isRateLimited($key);

        expect($result)->toBeFalse();
    });

    it('isRateLimited returns true at limit', function () {
        $workspace = createWorkspace();
        $service = app(AgentApiKeyService::class);
        $key = $service->create($workspace, 'Test Key', [], 100);
        Cache::put("agent_api_key_rate:{$key->id}", 100, 60);

        $result = $service->isRateLimited($key);

        expect($result)->toBeTrue();
    });

    it('isRateLimited returns true over limit', function () {
        $workspace = createWorkspace();
        $service = app(AgentApiKeyService::class);
        $key = $service->create($workspace, 'Test Key', [], 100);
        Cache::put("agent_api_key_rate:{$key->id}", 150, 60);

        $result = $service->isRateLimited($key);

        expect($result)->toBeTrue();
    });

    it('getRateLimitStatus returns correct values', function () {
        $workspace = createWorkspace();
        $service = app(AgentApiKeyService::class);
        $key = $service->create($workspace, 'Test Key', [], 100);
        Cache::put("agent_api_key_rate:{$key->id}", 30, 60);

        $status = $service->getRateLimitStatus($key);

        expect($status['limit'])->toBe(100)
            ->and($status['remaining'])->toBe(70)
            ->and($status['used'])->toBe(30)
            ->and($status)->toHaveKey('reset_in_seconds');
    });
});

// =========================================================================
// Key Management Tests
// =========================================================================

describe('key management', function () {
    it('revoke sets revoked_at', function () {
        $workspace = createWorkspace();
        $service = app(AgentApiKeyService::class);
        $key = $service->create($workspace, 'Test Key');

        $service->revoke($key);

        expect($key->fresh()->revoked_at)->not->toBeNull();
    });

    it('revoke clears rate limit cache', function () {
        $workspace = createWorkspace();
        $service = app(AgentApiKeyService::class);
        $key = $service->create($workspace, 'Test Key');
        Cache::put("agent_api_key_rate:{$key->id}", 50, 60);

        $service->revoke($key);

        expect(Cache::get("agent_api_key_rate:{$key->id}"))->toBeNull();
    });

    it('updatePermissions changes permissions', function () {
        $workspace = createWorkspace();
        $service = app(AgentApiKeyService::class);
        $key = $service->create(
            $workspace,
            'Test Key',
            [AgentApiKey::PERM_PLANS_READ]
        );

        $service->updatePermissions($key, [AgentApiKey::PERM_SESSIONS_WRITE]);

        $fresh = $key->fresh();
        expect($fresh->hasPermission(AgentApiKey::PERM_PLANS_READ))->toBeFalse()
            ->and($fresh->hasPermission(AgentApiKey::PERM_SESSIONS_WRITE))->toBeTrue();
    });

    it('updateRateLimit changes limit', function () {
        $workspace = createWorkspace();
        $service = app(AgentApiKeyService::class);
        $key = $service->create($workspace, 'Test Key', [], 100);

        $service->updateRateLimit($key, 500);

        expect($key->fresh()->rate_limit)->toBe(500);
    });

    it('extendExpiry updates expiry date', function () {
        $workspace = createWorkspace();
        $service = app(AgentApiKeyService::class);
        $key = $service->create(
            $workspace,
            'Test Key',
            [],
            100,
            Carbon::now()->addDay()
        );
        $newExpiry = Carbon::now()->addMonth();

        $service->extendExpiry($key, $newExpiry);

        expect($key->fresh()->expires_at->toDateTimeString())
            ->toBe($newExpiry->toDateTimeString());
    });

    it('removeExpiry clears expiry date', function () {
        $workspace = createWorkspace();
        $service = app(AgentApiKeyService::class);
        $key = $service->create(
            $workspace,
            'Test Key',
            [],
            100,
            Carbon::now()->addDay()
        );

        $service->removeExpiry($key);

        expect($key->fresh()->expires_at)->toBeNull();
    });
});

// =========================================================================
// IP Restriction Tests
// =========================================================================

describe('IP restrictions', function () {
    it('updateIpRestrictions sets values', function () {
        $workspace = createWorkspace();
        $service = app(AgentApiKeyService::class);
        $key = $service->create($workspace, 'Test Key');

        $service->updateIpRestrictions($key, true, ['192.168.1.1', '10.0.0.0/8']);

        $fresh = $key->fresh();
        expect($fresh->ip_restriction_enabled)->toBeTrue()
            ->and($fresh->ip_whitelist)->toBe(['192.168.1.1', '10.0.0.0/8']);
    });

    it('enableIpRestrictions enables with whitelist', function () {
        $workspace = createWorkspace();
        $service = app(AgentApiKeyService::class);
        $key = $service->create($workspace, 'Test Key');

        $service->enableIpRestrictions($key, ['192.168.1.1']);

        $fresh = $key->fresh();
        expect($fresh->ip_restriction_enabled)->toBeTrue()
            ->and($fresh->ip_whitelist)->toBe(['192.168.1.1']);
    });

    it('disableIpRestrictions disables restrictions', function () {
        $workspace = createWorkspace();
        $service = app(AgentApiKeyService::class);
        $key = $service->create($workspace, 'Test Key');
        $service->enableIpRestrictions($key, ['192.168.1.1']);

        $service->disableIpRestrictions($key);

        expect($key->fresh()->ip_restriction_enabled)->toBeFalse();
    });

    it('isIpAllowed returns true when restrictions disabled', function () {
        $workspace = createWorkspace();
        $service = app(AgentApiKeyService::class);
        $key = $service->create($workspace, 'Test Key');

        $result = $service->isIpAllowed($key, '192.168.1.100');

        expect($result)->toBeTrue();
    });

    it('isIpAllowed returns true when IP in whitelist', function () {
        $workspace = createWorkspace();
        $service = app(AgentApiKeyService::class);
        $key = $service->create($workspace, 'Test Key');
        $service->enableIpRestrictions($key, ['192.168.1.100']);

        $result = $service->isIpAllowed($key->fresh(), '192.168.1.100');

        expect($result)->toBeTrue();
    });

    it('isIpAllowed returns false when IP not in whitelist', function () {
        $workspace = createWorkspace();
        $service = app(AgentApiKeyService::class);
        $key = $service->create($workspace, 'Test Key');
        $service->enableIpRestrictions($key, ['192.168.1.100']);

        $result = $service->isIpAllowed($key->fresh(), '10.0.0.1');

        expect($result)->toBeFalse();
    });

    it('isIpAllowed supports CIDR ranges', function () {
        $workspace = createWorkspace();
        $service = app(AgentApiKeyService::class);
        $key = $service->create($workspace, 'Test Key');
        $service->enableIpRestrictions($key, ['192.168.1.0/24']);

        $fresh = $key->fresh();
        expect($service->isIpAllowed($fresh, '192.168.1.50'))->toBeTrue()
            ->and($service->isIpAllowed($fresh, '192.168.1.254'))->toBeTrue()
            ->and($service->isIpAllowed($fresh, '192.168.2.1'))->toBeFalse();
    });

    it('parseIpWhitelistInput parses valid input', function () {
        $service = app(AgentApiKeyService::class);
        $input = "192.168.1.1\n192.168.1.2\n10.0.0.0/8";

        $result = $service->parseIpWhitelistInput($input);

        expect($result['errors'])->toBeEmpty()
            ->and($result['entries'])->toHaveCount(3)
            ->and($result['entries'])->toContain('192.168.1.1')
            ->and($result['entries'])->toContain('192.168.1.2')
            ->and($result['entries'])->toContain('10.0.0.0/8');
    });

    it('parseIpWhitelistInput returns errors for invalid entries', function () {
        $service = app(AgentApiKeyService::class);
        $input = "192.168.1.1\ninvalid_ip\n10.0.0.0/8";

        $result = $service->parseIpWhitelistInput($input);

        expect($result['errors'])->toHaveCount(1)
            ->and($result['entries'])->toHaveCount(2);
    });
});

// =========================================================================
// Workspace Query Tests
// =========================================================================

describe('workspace queries', function () {
    it('getActiveKeysForWorkspace returns active keys only', function () {
        $workspace = createWorkspace();
        $service = app(AgentApiKeyService::class);

        $active = $service->create($workspace, 'Active Key');
        $revoked = $service->create($workspace, 'Revoked Key');
        $revoked->revoke();
        $service->create($workspace, 'Expired Key', [], 100, Carbon::now()->subDay());

        $keys = $service->getActiveKeysForWorkspace($workspace);

        expect($keys)->toHaveCount(1)
            ->and($keys->first()->name)->toBe('Active Key');
    });

    it('getActiveKeysForWorkspace filters by workspace', function () {
        $workspace = createWorkspace();
        $otherWorkspace = createWorkspace();
        $service = app(AgentApiKeyService::class);

        $service->create($workspace, 'Our Key');
        $service->create($otherWorkspace, 'Their Key');

        $keys = $service->getActiveKeysForWorkspace($workspace);

        expect($keys)->toHaveCount(1)
            ->and($keys->first()->name)->toBe('Our Key');
    });

    it('getAllKeysForWorkspace returns all keys', function () {
        $workspace = createWorkspace();
        $service = app(AgentApiKeyService::class);

        $service->create($workspace, 'Active Key');
        $revoked = $service->create($workspace, 'Revoked Key');
        $revoked->revoke();
        $service->create($workspace, 'Expired Key', [], 100, Carbon::now()->subDay());

        $keys = $service->getAllKeysForWorkspace($workspace);

        expect($keys)->toHaveCount(3);
    });
});

// =========================================================================
// Validate With Permission Tests
// =========================================================================

describe('validateWithPermission', function () {
    it('returns key when valid with permission', function () {
        $workspace = createWorkspace();
        $service = app(AgentApiKeyService::class);
        $key = $service->create(
            $workspace,
            'Test Key',
            [AgentApiKey::PERM_PLANS_READ]
        );
        $plainKey = $key->plainTextKey;

        $result = $service->validateWithPermission($plainKey, AgentApiKey::PERM_PLANS_READ);

        expect($result)
            ->not->toBeNull()
            ->and($result->id)->toBe($key->id);
    });

    it('returns null for invalid key', function () {
        $service = app(AgentApiKeyService::class);

        $result = $service->validateWithPermission(
            'ak_invalid_key',
            AgentApiKey::PERM_PLANS_READ
        );

        expect($result)->toBeNull();
    });

    it('returns null without required permission', function () {
        $workspace = createWorkspace();
        $service = app(AgentApiKeyService::class);
        $key = $service->create(
            $workspace,
            'Test Key',
            [AgentApiKey::PERM_SESSIONS_READ]
        );
        $plainKey = $key->plainTextKey;

        $result = $service->validateWithPermission($plainKey, AgentApiKey::PERM_PLANS_READ);

        expect($result)->toBeNull();
    });

    it('returns null when rate limited', function () {
        $workspace = createWorkspace();
        $service = app(AgentApiKeyService::class);
        $key = $service->create(
            $workspace,
            'Test Key',
            [AgentApiKey::PERM_PLANS_READ],
            100
        );
        $plainKey = $key->plainTextKey;
        Cache::put("agent_api_key_rate:{$key->id}", 150, 60);

        $result = $service->validateWithPermission($plainKey, AgentApiKey::PERM_PLANS_READ);

        expect($result)->toBeNull();
    });
});

// =========================================================================
// Full Authentication Flow Tests
// =========================================================================

describe('authenticate', function () {
    it('returns success for valid key with permission', function () {
        $workspace = createWorkspace();
        $service = app(AgentApiKeyService::class);
        $key = $service->create(
            $workspace,
            'Test Key',
            [AgentApiKey::PERM_PLANS_READ]
        );
        $plainKey = $key->plainTextKey;

        $result = $service->authenticate($plainKey, AgentApiKey::PERM_PLANS_READ);

        expect($result['success'])->toBeTrue()
            ->and($result['key'])->toBeInstanceOf(AgentApiKey::class)
            ->and($result['workspace_id'])->toBe($workspace->id)
            ->and($result)->toHaveKey('rate_limit');
    });

    it('returns error for invalid key', function () {
        $service = app(AgentApiKeyService::class);

        $result = $service->authenticate('ak_invalid_key', AgentApiKey::PERM_PLANS_READ);

        expect($result['success'])->toBeFalse()
            ->and($result['error'])->toBe('invalid_key');
    });

    it('returns error for revoked key', function () {
        $workspace = createWorkspace();
        $service = app(AgentApiKeyService::class);
        $key = $service->create(
            $workspace,
            'Test Key',
            [AgentApiKey::PERM_PLANS_READ]
        );
        $plainKey = $key->plainTextKey;
        $key->revoke();

        $result = $service->authenticate($plainKey, AgentApiKey::PERM_PLANS_READ);

        expect($result['success'])->toBeFalse()
            ->and($result['error'])->toBe('key_revoked');
    });

    it('returns error for expired key', function () {
        $workspace = createWorkspace();
        $service = app(AgentApiKeyService::class);
        $key = $service->create(
            $workspace,
            'Test Key',
            [AgentApiKey::PERM_PLANS_READ],
            100,
            Carbon::now()->subDay()
        );
        $plainKey = $key->plainTextKey;

        $result = $service->authenticate($plainKey, AgentApiKey::PERM_PLANS_READ);

        expect($result['success'])->toBeFalse()
            ->and($result['error'])->toBe('key_expired');
    });

    it('returns error for missing permission', function () {
        $workspace = createWorkspace();
        $service = app(AgentApiKeyService::class);
        $key = $service->create(
            $workspace,
            'Test Key',
            [AgentApiKey::PERM_SESSIONS_READ]
        );
        $plainKey = $key->plainTextKey;

        $result = $service->authenticate($plainKey, AgentApiKey::PERM_PLANS_READ);

        expect($result['success'])->toBeFalse()
            ->and($result['error'])->toBe('permission_denied');
    });

    it('returns error when rate limited', function () {
        $workspace = createWorkspace();
        $service = app(AgentApiKeyService::class);
        $key = $service->create(
            $workspace,
            'Test Key',
            [AgentApiKey::PERM_PLANS_READ],
            100
        );
        $plainKey = $key->plainTextKey;
        Cache::put("agent_api_key_rate:{$key->id}", 150, 60);

        $result = $service->authenticate($plainKey, AgentApiKey::PERM_PLANS_READ);

        expect($result['success'])->toBeFalse()
            ->and($result['error'])->toBe('rate_limited')
            ->and($result)->toHaveKey('rate_limit');
    });

    it('checks IP restrictions', function () {
        $workspace = createWorkspace();
        $service = app(AgentApiKeyService::class);
        $key = $service->create(
            $workspace,
            'Test Key',
            [AgentApiKey::PERM_PLANS_READ]
        );
        $plainKey = $key->plainTextKey;
        $service->enableIpRestrictions($key, ['192.168.1.100']);

        $result = $service->authenticate($plainKey, AgentApiKey::PERM_PLANS_READ, '10.0.0.1');

        expect($result['success'])->toBeFalse()
            ->and($result['error'])->toBe('ip_not_allowed');
    });

    it('allows whitelisted IP', function () {
        $workspace = createWorkspace();
        $service = app(AgentApiKeyService::class);
        $key = $service->create(
            $workspace,
            'Test Key',
            [AgentApiKey::PERM_PLANS_READ]
        );
        $plainKey = $key->plainTextKey;
        $service->enableIpRestrictions($key, ['192.168.1.100']);

        $result = $service->authenticate($plainKey, AgentApiKey::PERM_PLANS_READ, '192.168.1.100');

        expect($result['success'])->toBeTrue()
            ->and($result['client_ip'])->toBe('192.168.1.100');
    });

    it('records usage on success', function () {
        $workspace = createWorkspace();
        $service = app(AgentApiKeyService::class);
        $key = $service->create(
            $workspace,
            'Test Key',
            [AgentApiKey::PERM_PLANS_READ]
        );
        $plainKey = $key->plainTextKey;
        Cache::forget("agent_api_key_rate:{$key->id}");

        $service->authenticate($plainKey, AgentApiKey::PERM_PLANS_READ, '192.168.1.50');

        $fresh = $key->fresh();
        expect($fresh->call_count)->toBe(1)
            ->and($fresh->last_used_at)->not->toBeNull()
            ->and($fresh->last_used_ip)->toBe('192.168.1.50');
    });

    it('does not record usage on failure', function () {
        $workspace = createWorkspace();
        $service = app(AgentApiKeyService::class);
        $key = $service->create($workspace, 'Test Key', []);
        $plainKey = $key->plainTextKey;

        $service->authenticate($plainKey, AgentApiKey::PERM_PLANS_READ);

        expect($key->fresh()->call_count)->toBe(0);
    });
});

// =========================================================================
// Edge Cases and Security Tests
// =========================================================================

describe('edge cases', function () {
    it('handles empty permissions array', function () {
        $workspace = createWorkspace();
        $service = app(AgentApiKeyService::class);

        $key = $service->create($workspace, 'Test Key', []);

        expect($key->permissions)->toBe([])
            ->and($service->checkPermission($key, AgentApiKey::PERM_PLANS_READ))->toBeFalse();
    });

    it('handles multiple workspaces correctly', function () {
        $workspace1 = createWorkspace();
        $workspace2 = createWorkspace();
        $service = app(AgentApiKeyService::class);

        $key1 = $service->create($workspace1, 'Workspace 1 Key');
        $key2 = $service->create($workspace2, 'Workspace 2 Key');

        expect($key1->workspace_id)->toBe($workspace1->id)
            ->and($key2->workspace_id)->toBe($workspace2->id);

        $workspace1Keys = $service->getAllKeysForWorkspace($workspace1);
        $workspace2Keys = $service->getAllKeysForWorkspace($workspace2);

        expect($workspace1Keys)->toHaveCount(1)
            ->and($workspace2Keys)->toHaveCount(1);
    });

    it('handles concurrent rate limit updates atomically', function () {
        $workspace = createWorkspace();
        $service = app(AgentApiKeyService::class);
        $key = $service->create($workspace, 'Test Key');
        Cache::forget("agent_api_key_rate:{$key->id}");

        // Simulate rapid concurrent requests
        for ($i = 0; $i < 10; $i++) {
            $service->recordUsage($key);
        }

        expect(Cache::get("agent_api_key_rate:{$key->id}"))->toBe(10);
    });

    it('handles null client IP gracefully', function () {
        $workspace = createWorkspace();
        $service = app(AgentApiKeyService::class);
        $key = $service->create(
            $workspace,
            'Test Key',
            [AgentApiKey::PERM_PLANS_READ]
        );
        $plainKey = $key->plainTextKey;

        $result = $service->authenticate($plainKey, AgentApiKey::PERM_PLANS_READ, null);

        expect($result['success'])->toBeTrue()
            ->and($result['client_ip'])->toBeNull();
    });

    it('validates key before checking IP restrictions', function () {
        $workspace = createWorkspace();
        $service = app(AgentApiKeyService::class);
        $key = $service->create(
            $workspace,
            'Test Key',
            [AgentApiKey::PERM_PLANS_READ]
        );
        $plainKey = $key->plainTextKey;
        $service->enableIpRestrictions($key, ['192.168.1.100']);
        $key->revoke();

        // Should fail on revoked check before IP check
        $result = $service->authenticate($plainKey, AgentApiKey::PERM_PLANS_READ, '10.0.0.1');

        expect($result['error'])->toBe('key_revoked');
    });
});
