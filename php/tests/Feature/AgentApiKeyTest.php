<?php

declare(strict_types=1);

/**
 * Tests for the AgentApiKey model.
 *
 * Covers generation, validation, permissions, rate limiting, and IP restrictions.
 */

use Carbon\Carbon;
use Core\Mod\Agentic\Models\AgentApiKey;
use Core\Tenant\Models\Workspace;
use Illuminate\Support\Facades\Cache;

// =========================================================================
// Key Generation Tests
// =========================================================================

describe('key generation', function () {
    it('generates key with correct prefix', function () {
        $workspace = createWorkspace();
        $key = AgentApiKey::generate($workspace, 'Test Key');

        expect($key->plainTextKey)
            ->toStartWith('ak_')
            ->toHaveLength(35); // ak_ + 32 random chars
    });

    it('stores hashed key with Argon2id', function () {
        $workspace = createWorkspace();
        $key = AgentApiKey::generate($workspace, 'Test Key');

        // Argon2id hashes start with $argon2id$
        expect($key->key)->toStartWith('$argon2id$');
    });

    it('makes plaintext key available only once after creation', function () {
        $workspace = createWorkspace();
        $key = AgentApiKey::generate($workspace, 'Test Key');

        expect($key->plainTextKey)->not->toBeNull();

        // After fetching from database, plaintext should be null
        $freshKey = AgentApiKey::find($key->id);

        expect($freshKey->plainTextKey)->toBeNull();
    });

    it('generates key with workspace ID', function () {
        $workspace = createWorkspace();
        $key = AgentApiKey::generate($workspace->id, 'Test Key');

        expect($key->workspace_id)->toBe($workspace->id);
    });

    it('generates key with workspace model', function () {
        $workspace = createWorkspace();
        $key = AgentApiKey::generate($workspace, 'Test Key');

        expect($key->workspace_id)->toBe($workspace->id);
    });

    it('generates key with permissions', function () {
        $workspace = createWorkspace();
        $permissions = [
            AgentApiKey::PERM_PLANS_READ,
            AgentApiKey::PERM_PLANS_WRITE,
        ];

        $key = AgentApiKey::generate($workspace, 'Test Key', $permissions);

        expect($key->permissions)->toBe($permissions);
    });

    it('generates key with custom rate limit', function () {
        $workspace = createWorkspace();
        $key = AgentApiKey::generate($workspace, 'Test Key', [], 500);

        expect($key->rate_limit)->toBe(500);
    });

    it('generates key with expiry date', function () {
        $workspace = createWorkspace();
        $expiresAt = Carbon::now()->addDays(30);

        $key = AgentApiKey::generate($workspace, 'Test Key', [], 100, $expiresAt);

        expect($key->expires_at->toDateTimeString())
            ->toBe($expiresAt->toDateTimeString());
    });

    it('initialises call count to zero', function () {
        $workspace = createWorkspace();
        $key = AgentApiKey::generate($workspace, 'Test Key');

        expect($key->call_count)->toBe(0);
    });
});

// =========================================================================
// Key Lookup Tests
// =========================================================================

describe('key lookup', function () {
    it('finds key by plaintext value', function () {
        $workspace = createWorkspace();
        $key = AgentApiKey::generate($workspace, 'Test Key');
        $plainKey = $key->plainTextKey;

        $found = AgentApiKey::findByKey($plainKey);

        expect($found)
            ->not->toBeNull()
            ->and($found->id)->toBe($key->id);
    });

    it('returns null for invalid key', function () {
        $workspace = createWorkspace();
        AgentApiKey::generate($workspace, 'Test Key');

        $found = AgentApiKey::findByKey('ak_invalid_key_that_does_not_exist');

        expect($found)->toBeNull();
    });

    it('returns null for malformed key', function () {
        expect(AgentApiKey::findByKey(''))->toBeNull();
        expect(AgentApiKey::findByKey('invalid'))->toBeNull();
        expect(AgentApiKey::findByKey('ak_short'))->toBeNull();
    });

    it('does not find revoked keys', function () {
        $workspace = createWorkspace();
        $key = AgentApiKey::generate($workspace, 'Test Key');
        $plainKey = $key->plainTextKey;

        $key->revoke();

        $found = AgentApiKey::findByKey($plainKey);

        expect($found)->toBeNull();
    });

    it('does not find expired keys', function () {
        $workspace = createWorkspace();
        $key = AgentApiKey::generate(
            $workspace,
            'Test Key',
            [],
            100,
            Carbon::now()->subDay()
        );
        $plainKey = $key->plainTextKey;

        $found = AgentApiKey::findByKey($plainKey);

        expect($found)->toBeNull();
    });
});

// =========================================================================
// Key Verification Tests
// =========================================================================

describe('key verification', function () {
    it('verifyKey returns true for matching key', function () {
        $workspace = createWorkspace();
        $key = AgentApiKey::generate($workspace, 'Test Key');
        $plainKey = $key->plainTextKey;

        expect($key->verifyKey($plainKey))->toBeTrue();
    });

    it('verifyKey returns false for non-matching key', function () {
        $workspace = createWorkspace();
        $key = AgentApiKey::generate($workspace, 'Test Key');

        expect($key->verifyKey('ak_wrong_key_entirely'))->toBeFalse();
    });
});

// =========================================================================
// Status Tests
// =========================================================================

describe('status helpers', function () {
    it('isActive returns true for fresh key', function () {
        $workspace = createWorkspace();
        $key = AgentApiKey::generate($workspace, 'Test Key');

        expect($key->isActive())->toBeTrue();
    });

    it('isActive returns false for revoked key', function () {
        $workspace = createWorkspace();
        $key = AgentApiKey::generate($workspace, 'Test Key');
        $key->revoke();

        expect($key->isActive())->toBeFalse();
    });

    it('isActive returns false for expired key', function () {
        $workspace = createWorkspace();
        $key = AgentApiKey::generate(
            $workspace,
            'Test Key',
            [],
            100,
            Carbon::now()->subDay()
        );

        expect($key->isActive())->toBeFalse();
    });

    it('isActive returns true for key with future expiry', function () {
        $workspace = createWorkspace();
        $key = AgentApiKey::generate(
            $workspace,
            'Test Key',
            [],
            100,
            Carbon::now()->addDay()
        );

        expect($key->isActive())->toBeTrue();
    });

    it('isRevoked returns correct value', function () {
        $workspace = createWorkspace();
        $key = AgentApiKey::generate($workspace, 'Test Key');

        expect($key->isRevoked())->toBeFalse();

        $key->revoke();

        expect($key->isRevoked())->toBeTrue();
    });

    it('isExpired returns correct value for various states', function () {
        $workspace = createWorkspace();

        $notExpired = AgentApiKey::generate(
            $workspace,
            'Not Expired',
            [],
            100,
            Carbon::now()->addDay()
        );

        $expired = AgentApiKey::generate(
            $workspace,
            'Expired',
            [],
            100,
            Carbon::now()->subDay()
        );

        $noExpiry = AgentApiKey::generate($workspace, 'No Expiry');

        expect($notExpired->isExpired())->toBeFalse();
        expect($expired->isExpired())->toBeTrue();
        expect($noExpiry->isExpired())->toBeFalse();
    });
});

// =========================================================================
// Permission Tests
// =========================================================================

describe('permissions', function () {
    it('hasPermission returns true when granted', function () {
        $workspace = createWorkspace();
        $key = AgentApiKey::generate(
            $workspace,
            'Test Key',
            [AgentApiKey::PERM_PLANS_READ, AgentApiKey::PERM_PLANS_WRITE]
        );

        expect($key->hasPermission(AgentApiKey::PERM_PLANS_READ))->toBeTrue();
        expect($key->hasPermission(AgentApiKey::PERM_PLANS_WRITE))->toBeTrue();
    });

    it('hasPermission returns false when not granted', function () {
        $workspace = createWorkspace();
        $key = AgentApiKey::generate(
            $workspace,
            'Test Key',
            [AgentApiKey::PERM_PLANS_READ]
        );

        expect($key->hasPermission(AgentApiKey::PERM_PLANS_WRITE))->toBeFalse();
    });

    it('hasAnyPermission returns true when one matches', function () {
        $workspace = createWorkspace();
        $key = AgentApiKey::generate(
            $workspace,
            'Test Key',
            [AgentApiKey::PERM_PLANS_READ]
        );

        expect($key->hasAnyPermission([
            AgentApiKey::PERM_PLANS_READ,
            AgentApiKey::PERM_PLANS_WRITE,
        ]))->toBeTrue();
    });

    it('hasAnyPermission returns false when none match', function () {
        $workspace = createWorkspace();
        $key = AgentApiKey::generate(
            $workspace,
            'Test Key',
            [AgentApiKey::PERM_TEMPLATES_READ]
        );

        expect($key->hasAnyPermission([
            AgentApiKey::PERM_PLANS_READ,
            AgentApiKey::PERM_PLANS_WRITE,
        ]))->toBeFalse();
    });

    it('hasAllPermissions returns true when all granted', function () {
        $workspace = createWorkspace();
        $key = AgentApiKey::generate(
            $workspace,
            'Test Key',
            [AgentApiKey::PERM_PLANS_READ, AgentApiKey::PERM_PLANS_WRITE, AgentApiKey::PERM_SESSIONS_READ]
        );

        expect($key->hasAllPermissions([
            AgentApiKey::PERM_PLANS_READ,
            AgentApiKey::PERM_PLANS_WRITE,
        ]))->toBeTrue();
    });

    it('hasAllPermissions returns false when missing one', function () {
        $workspace = createWorkspace();
        $key = AgentApiKey::generate(
            $workspace,
            'Test Key',
            [AgentApiKey::PERM_PLANS_READ]
        );

        expect($key->hasAllPermissions([
            AgentApiKey::PERM_PLANS_READ,
            AgentApiKey::PERM_PLANS_WRITE,
        ]))->toBeFalse();
    });

    it('updatePermissions changes permissions', function () {
        $workspace = createWorkspace();
        $key = AgentApiKey::generate(
            $workspace,
            'Test Key',
            [AgentApiKey::PERM_PLANS_READ]
        );

        $key->updatePermissions([AgentApiKey::PERM_SESSIONS_WRITE]);

        expect($key->hasPermission(AgentApiKey::PERM_PLANS_READ))->toBeFalse();
        expect($key->hasPermission(AgentApiKey::PERM_SESSIONS_WRITE))->toBeTrue();
    });
});

// =========================================================================
// Rate Limiting Tests
// =========================================================================

describe('rate limiting', function () {
    it('isRateLimited returns false when under limit', function () {
        $workspace = createWorkspace();
        $key = AgentApiKey::generate($workspace, 'Test Key', [], 100);

        Cache::put("agent_api_key_rate:{$key->id}", 50, 60);

        expect($key->isRateLimited())->toBeFalse();
    });

    it('isRateLimited returns true when at limit', function () {
        $workspace = createWorkspace();
        $key = AgentApiKey::generate($workspace, 'Test Key', [], 100);

        Cache::put("agent_api_key_rate:{$key->id}", 100, 60);

        expect($key->isRateLimited())->toBeTrue();
    });

    it('isRateLimited returns true when over limit', function () {
        $workspace = createWorkspace();
        $key = AgentApiKey::generate($workspace, 'Test Key', [], 100);

        Cache::put("agent_api_key_rate:{$key->id}", 150, 60);

        expect($key->isRateLimited())->toBeTrue();
    });

    it('getRecentCallCount returns cache value', function () {
        $workspace = createWorkspace();
        $key = AgentApiKey::generate($workspace, 'Test Key');

        Cache::put("agent_api_key_rate:{$key->id}", 42, 60);

        expect($key->getRecentCallCount())->toBe(42);
    });

    it('getRecentCallCount returns zero when not cached', function () {
        $workspace = createWorkspace();
        $key = AgentApiKey::generate($workspace, 'Test Key');

        expect($key->getRecentCallCount())->toBe(0);
    });

    it('getRemainingCalls returns correct value', function () {
        $workspace = createWorkspace();
        $key = AgentApiKey::generate($workspace, 'Test Key', [], 100);

        Cache::put("agent_api_key_rate:{$key->id}", 30, 60);

        expect($key->getRemainingCalls())->toBe(70);
    });

    it('getRemainingCalls returns zero when over limit', function () {
        $workspace = createWorkspace();
        $key = AgentApiKey::generate($workspace, 'Test Key', [], 100);

        Cache::put("agent_api_key_rate:{$key->id}", 150, 60);

        expect($key->getRemainingCalls())->toBe(0);
    });

    it('updateRateLimit changes limit', function () {
        $workspace = createWorkspace();
        $key = AgentApiKey::generate($workspace, 'Test Key', [], 100);

        $key->updateRateLimit(200);

        expect($key->fresh()->rate_limit)->toBe(200);
    });
});

// =========================================================================
// IP Restriction Tests
// =========================================================================

describe('IP restrictions', function () {
    it('has IP restrictions disabled by default', function () {
        $workspace = createWorkspace();
        $key = AgentApiKey::generate($workspace, 'Test Key');

        expect($key->ip_restriction_enabled)->toBeFalse();
    });

    it('enableIpRestriction sets flag', function () {
        $workspace = createWorkspace();
        $key = AgentApiKey::generate($workspace, 'Test Key');

        $key->enableIpRestriction();

        expect($key->fresh()->ip_restriction_enabled)->toBeTrue();
    });

    it('disableIpRestriction clears flag', function () {
        $workspace = createWorkspace();
        $key = AgentApiKey::generate($workspace, 'Test Key');
        $key->enableIpRestriction();

        $key->disableIpRestriction();

        expect($key->fresh()->ip_restriction_enabled)->toBeFalse();
    });

    it('updateIpWhitelist sets list', function () {
        $workspace = createWorkspace();
        $key = AgentApiKey::generate($workspace, 'Test Key');

        $key->updateIpWhitelist(['192.168.1.1', '10.0.0.0/8']);

        expect($key->fresh()->ip_whitelist)->toBe(['192.168.1.1', '10.0.0.0/8']);
    });

    it('addToIpWhitelist adds entry', function () {
        $workspace = createWorkspace();
        $key = AgentApiKey::generate($workspace, 'Test Key');
        $key->updateIpWhitelist(['192.168.1.1']);

        $key->addToIpWhitelist('10.0.0.1');

        $whitelist = $key->fresh()->ip_whitelist;
        expect($whitelist)->toContain('192.168.1.1');
        expect($whitelist)->toContain('10.0.0.1');
    });

    it('addToIpWhitelist does not duplicate', function () {
        $workspace = createWorkspace();
        $key = AgentApiKey::generate($workspace, 'Test Key');
        $key->updateIpWhitelist(['192.168.1.1']);

        $key->addToIpWhitelist('192.168.1.1');

        expect($key->fresh()->ip_whitelist)->toHaveCount(1);
    });

    it('removeFromIpWhitelist removes entry', function () {
        $workspace = createWorkspace();
        $key = AgentApiKey::generate($workspace, 'Test Key');
        $key->updateIpWhitelist(['192.168.1.1', '10.0.0.1']);

        $key->removeFromIpWhitelist('192.168.1.1');

        $whitelist = $key->fresh()->ip_whitelist;
        expect($whitelist)->not->toContain('192.168.1.1');
        expect($whitelist)->toContain('10.0.0.1');
    });

    it('hasIpRestrictions returns correct value', function () {
        $workspace = createWorkspace();
        $key = AgentApiKey::generate($workspace, 'Test Key');

        // No restrictions
        expect($key->hasIpRestrictions())->toBeFalse();

        // Enabled but no whitelist
        $key->enableIpRestriction();
        expect($key->fresh()->hasIpRestrictions())->toBeFalse();

        // Enabled with whitelist
        $key->updateIpWhitelist(['192.168.1.1']);
        expect($key->fresh()->hasIpRestrictions())->toBeTrue();
    });

    it('getIpWhitelistCount returns correct value', function () {
        $workspace = createWorkspace();
        $key = AgentApiKey::generate($workspace, 'Test Key');

        expect($key->getIpWhitelistCount())->toBe(0);

        $key->updateIpWhitelist(['192.168.1.1', '10.0.0.0/8', '172.16.0.0/12']);

        expect($key->fresh()->getIpWhitelistCount())->toBe(3);
    });

    it('recordLastUsedIp stores IP', function () {
        $workspace = createWorkspace();
        $key = AgentApiKey::generate($workspace, 'Test Key');

        $key->recordLastUsedIp('192.168.1.100');

        expect($key->fresh()->last_used_ip)->toBe('192.168.1.100');
    });
});

// =========================================================================
// Actions Tests
// =========================================================================

describe('actions', function () {
    it('revoke sets revoked_at', function () {
        $workspace = createWorkspace();
        $key = AgentApiKey::generate($workspace, 'Test Key');

        $key->revoke();

        expect($key->fresh()->revoked_at)->not->toBeNull();
    });

    it('recordUsage increments count', function () {
        $workspace = createWorkspace();
        $key = AgentApiKey::generate($workspace, 'Test Key');

        $key->recordUsage();
        $key->recordUsage();
        $key->recordUsage();

        expect($key->fresh()->call_count)->toBe(3);
    });

    it('recordUsage updates last_used_at', function () {
        $workspace = createWorkspace();
        $key = AgentApiKey::generate($workspace, 'Test Key');

        expect($key->last_used_at)->toBeNull();

        $key->recordUsage();

        expect($key->fresh()->last_used_at)->not->toBeNull();
    });

    it('extendExpiry updates expiry', function () {
        $workspace = createWorkspace();
        $key = AgentApiKey::generate(
            $workspace,
            'Test Key',
            [],
            100,
            Carbon::now()->addDay()
        );

        $newExpiry = Carbon::now()->addMonth();
        $key->extendExpiry($newExpiry);

        expect($key->fresh()->expires_at->toDateTimeString())
            ->toBe($newExpiry->toDateTimeString());
    });

    it('removeExpiry clears expiry', function () {
        $workspace = createWorkspace();
        $key = AgentApiKey::generate(
            $workspace,
            'Test Key',
            [],
            100,
            Carbon::now()->addDay()
        );

        $key->removeExpiry();

        expect($key->fresh()->expires_at)->toBeNull();
    });
});

// =========================================================================
// Scope Tests
// =========================================================================

describe('scopes', function () {
    it('active scope filters correctly', function () {
        $workspace = createWorkspace();

        AgentApiKey::generate($workspace, 'Active Key');
        $revoked = AgentApiKey::generate($workspace, 'Revoked Key');
        $revoked->revoke();
        AgentApiKey::generate($workspace, 'Expired Key', [], 100, Carbon::now()->subDay());

        $activeKeys = AgentApiKey::active()->get();

        expect($activeKeys)->toHaveCount(1);
        expect($activeKeys->first()->name)->toBe('Active Key');
    });

    it('forWorkspace scope filters correctly', function () {
        $workspace = createWorkspace();
        $otherWorkspace = createWorkspace();

        AgentApiKey::generate($workspace, 'Our Key');
        AgentApiKey::generate($otherWorkspace, 'Their Key');

        $ourKeys = AgentApiKey::forWorkspace($workspace)->get();

        expect($ourKeys)->toHaveCount(1);
        expect($ourKeys->first()->name)->toBe('Our Key');
    });

    it('revoked scope filters correctly', function () {
        $workspace = createWorkspace();

        AgentApiKey::generate($workspace, 'Active Key');
        $revoked = AgentApiKey::generate($workspace, 'Revoked Key');
        $revoked->revoke();

        $revokedKeys = AgentApiKey::revoked()->get();

        expect($revokedKeys)->toHaveCount(1);
        expect($revokedKeys->first()->name)->toBe('Revoked Key');
    });

    it('expired scope filters correctly', function () {
        $workspace = createWorkspace();

        AgentApiKey::generate($workspace, 'Active Key');
        AgentApiKey::generate($workspace, 'Expired Key', [], 100, Carbon::now()->subDay());

        $expiredKeys = AgentApiKey::expired()->get();

        expect($expiredKeys)->toHaveCount(1);
        expect($expiredKeys->first()->name)->toBe('Expired Key');
    });
});

// =========================================================================
// Display Helper Tests
// =========================================================================

describe('display helpers', function () {
    it('getMaskedKey returns masked format', function () {
        $workspace = createWorkspace();
        $key = AgentApiKey::generate($workspace, 'Test Key');

        $masked = $key->getMaskedKey();

        expect($masked)
            ->toStartWith('ak_')
            ->toEndWith('...');
    });

    it('getStatusLabel returns correct label', function () {
        $workspace = createWorkspace();

        $active = AgentApiKey::generate($workspace, 'Active');
        $revoked = AgentApiKey::generate($workspace, 'Revoked');
        $revoked->revoke();
        $expired = AgentApiKey::generate($workspace, 'Expired', [], 100, Carbon::now()->subDay());

        expect($active->getStatusLabel())->toBe('Active');
        expect($revoked->getStatusLabel())->toBe('Revoked');
        expect($expired->getStatusLabel())->toBe('Expired');
    });

    it('getStatusColor returns correct colour', function () {
        $workspace = createWorkspace();

        $active = AgentApiKey::generate($workspace, 'Active');
        $revoked = AgentApiKey::generate($workspace, 'Revoked');
        $revoked->revoke();
        $expired = AgentApiKey::generate($workspace, 'Expired', [], 100, Carbon::now()->subDay());

        expect($active->getStatusColor())->toBe('green');
        expect($revoked->getStatusColor())->toBe('red');
        expect($expired->getStatusColor())->toBe('amber');
    });

    it('getLastUsedForHumans returns Never when null', function () {
        $workspace = createWorkspace();
        $key = AgentApiKey::generate($workspace, 'Test Key');

        expect($key->getLastUsedForHumans())->toBe('Never');
    });

    it('getLastUsedForHumans returns diff when set', function () {
        $workspace = createWorkspace();
        $key = AgentApiKey::generate($workspace, 'Test Key');
        $key->update(['last_used_at' => Carbon::now()->subHour()]);

        expect($key->getLastUsedForHumans())->toContain('ago');
    });

    it('getExpiresForHumans returns Never when null', function () {
        $workspace = createWorkspace();
        $key = AgentApiKey::generate($workspace, 'Test Key');

        expect($key->getExpiresForHumans())->toBe('Never');
    });

    it('getExpiresForHumans returns Expired when past', function () {
        $workspace = createWorkspace();
        $key = AgentApiKey::generate(
            $workspace,
            'Test Key',
            [],
            100,
            Carbon::now()->subDay()
        );

        expect($key->getExpiresForHumans())->toContain('Expired');
    });

    it('getExpiresForHumans returns Expires when future', function () {
        $workspace = createWorkspace();
        $key = AgentApiKey::generate(
            $workspace,
            'Test Key',
            [],
            100,
            Carbon::now()->addDay()
        );

        expect($key->getExpiresForHumans())->toContain('Expires');
    });
});

// =========================================================================
// Array Output Tests
// =========================================================================

describe('array output', function () {
    it('toArray includes expected keys', function () {
        $workspace = createWorkspace();
        $key = AgentApiKey::generate(
            $workspace,
            'Test Key',
            [AgentApiKey::PERM_PLANS_READ]
        );

        $array = $key->toArray();

        expect($array)
            ->toHaveKey('id')
            ->toHaveKey('workspace_id')
            ->toHaveKey('name')
            ->toHaveKey('permissions')
            ->toHaveKey('rate_limit')
            ->toHaveKey('call_count')
            ->toHaveKey('status')
            ->toHaveKey('ip_restriction_enabled')
            ->toHaveKey('ip_whitelist_count');

        // Should NOT include the key hash
        expect($array)->not->toHaveKey('key');
    });
});

// =========================================================================
// Available Permissions Tests
// =========================================================================

describe('available permissions', function () {
    it('returns all permissions', function () {
        $permissions = AgentApiKey::availablePermissions();

        expect($permissions)
            ->toBeArray()
            ->toHaveKey(AgentApiKey::PERM_PLANS_READ)
            ->toHaveKey(AgentApiKey::PERM_PLANS_WRITE)
            ->toHaveKey(AgentApiKey::PERM_SESSIONS_READ)
            ->toHaveKey(AgentApiKey::PERM_SESSIONS_WRITE)
            ->toHaveKey(AgentApiKey::PERM_NOTIFY_READ)
            ->toHaveKey(AgentApiKey::PERM_NOTIFY_WRITE)
            ->toHaveKey(AgentApiKey::PERM_NOTIFY_SEND);
    });
});

// =========================================================================
// Relationship Tests
// =========================================================================

describe('relationships', function () {
    it('belongs to workspace', function () {
        $workspace = createWorkspace();
        $key = AgentApiKey::generate($workspace, 'Test Key');

        expect($key->workspace)
            ->toBeInstanceOf(Workspace::class)
            ->and($key->workspace->id)->toBe($workspace->id);
    });
});

// =========================================================================
// Key Rotation Tests
// =========================================================================

describe('key rotation', function () {
    it('can generate a new key for the same workspace and revoke old', function () {
        $workspace = createWorkspace();

        $oldKey = AgentApiKey::generate($workspace, 'Old Key');
        $oldPlainKey = $oldKey->plainTextKey;

        // Revoke old key
        $oldKey->revoke();

        // Create new key
        $newKey = AgentApiKey::generate($workspace, 'New Key');
        $newPlainKey = $newKey->plainTextKey;

        // Old key should not be found
        expect(AgentApiKey::findByKey($oldPlainKey))->toBeNull();

        // New key should be found
        expect(AgentApiKey::findByKey($newPlainKey))->not->toBeNull();
    });

    it('workspace can have multiple active keys', function () {
        $workspace = createWorkspace();

        AgentApiKey::generate($workspace, 'Key 1');
        AgentApiKey::generate($workspace, 'Key 2');
        AgentApiKey::generate($workspace, 'Key 3');

        $activeKeys = AgentApiKey::forWorkspace($workspace)->active()->get();

        expect($activeKeys)->toHaveCount(3);
    });
});

// =========================================================================
// Security Edge Cases
// =========================================================================

describe('security edge cases', function () {
    it('different keys for same workspace have unique hashes', function () {
        $workspace = createWorkspace();

        $key1 = AgentApiKey::generate($workspace, 'Key 1');
        $key2 = AgentApiKey::generate($workspace, 'Key 2');

        expect($key1->key)->not->toBe($key2->key);
    });

    it('same plaintext would produce different Argon2id hashes', function () {
        // This tests that Argon2id includes a random salt
        $workspace = createWorkspace();
        $plainKey = 'ak_test_key_12345678901234567890';

        $hash1 = password_hash($plainKey, PASSWORD_ARGON2ID);
        $hash2 = password_hash($plainKey, PASSWORD_ARGON2ID);

        expect($hash1)->not->toBe($hash2);

        // But both verify correctly
        expect(password_verify($plainKey, $hash1))->toBeTrue();
        expect(password_verify($plainKey, $hash2))->toBeTrue();
    });

    it('cannot find key using partial plaintext', function () {
        $workspace = createWorkspace();
        $key = AgentApiKey::generate($workspace, 'Test Key');
        $partialKey = substr($key->plainTextKey, 0, 20);

        expect(AgentApiKey::findByKey($partialKey))->toBeNull();
    });

    it('cannot find key using hash directly', function () {
        $workspace = createWorkspace();
        $key = AgentApiKey::generate($workspace, 'Test Key');

        // Trying to use the hash as if it were the plaintext key
        expect(AgentApiKey::findByKey($key->key))->toBeNull();
    });
});
