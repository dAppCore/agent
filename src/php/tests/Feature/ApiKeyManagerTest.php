<?php

declare(strict_types=1);

/**
 * Integration tests for ApiKeyManager admin UI component.
 *
 * Verifies that ApiKeyManager consistently uses AgentApiKey model
 * for all create, list, and revoke operations.
 */

use Core\Mod\Agentic\Models\AgentApiKey;
use Core\Mod\Agentic\Services\AgentApiKeyService;
use Core\Tenant\Models\Workspace;

// =========================================================================
// Model Consistency Tests
// =========================================================================

describe('ApiKeyManager model consistency', function () {
    it('ApiKeyManager uses AgentApiKey class', function () {
        $source = file_get_contents(__DIR__.'/../../View/Modal/Admin/ApiKeyManager.php');

        expect($source)
            ->toContain('Core\Mod\Agentic\Models\AgentApiKey')
            ->not->toContain('Core\Api\Models\ApiKey')
            ->not->toContain('Core\Api\ApiKey');
    });

    it('ApiKeyManager uses AgentApiKeyService', function () {
        $source = file_get_contents(__DIR__.'/../../View/Modal/Admin/ApiKeyManager.php');

        expect($source)->toContain('Core\Mod\Agentic\Services\AgentApiKeyService');
    });

    it('ApiKeyManager does not reference old scopes property', function () {
        $source = file_get_contents(__DIR__.'/../../View/Modal/Admin/ApiKeyManager.php');

        expect($source)
            ->not->toContain('newKeyScopes')
            ->not->toContain('toggleScope');
    });

    it('blade template uses permissions not scopes', function () {
        $source = file_get_contents(__DIR__.'/../../View/Blade/admin/api-key-manager.blade.php');

        expect($source)
            ->toContain('$key->permissions')
            ->not->toContain('$key->scopes');
    });

    it('blade template uses getMaskedKey not prefix', function () {
        $source = file_get_contents(__DIR__.'/../../View/Blade/admin/api-key-manager.blade.php');

        expect($source)
            ->toContain('getMaskedKey()')
            ->not->toContain('$key->prefix');
    });

    it('blade template calls togglePermission not toggleScope', function () {
        $source = file_get_contents(__DIR__.'/../../View/Blade/admin/api-key-manager.blade.php');

        expect($source)
            ->toContain('togglePermission')
            ->not->toContain('toggleScope');
    });
});

// =========================================================================
// AgentApiKey Integration Tests (via service, as used by ApiKeyManager)
// =========================================================================

describe('ApiKeyManager key creation integration', function () {
    it('creates an AgentApiKey via service', function () {
        $workspace = createWorkspace();
        $service = app(AgentApiKeyService::class);

        $key = $service->create(
            workspace: $workspace,
            name: 'Workspace MCP Key',
            permissions: [AgentApiKey::PERM_PLANS_READ, AgentApiKey::PERM_SESSIONS_READ],
        );

        expect($key)->toBeInstanceOf(AgentApiKey::class)
            ->and($key->name)->toBe('Workspace MCP Key')
            ->and($key->workspace_id)->toBe($workspace->id)
            ->and($key->permissions)->toContain(AgentApiKey::PERM_PLANS_READ)
            ->and($key->plainTextKey)->toStartWith('ak_');
    });

    it('plain text key is only available once after creation', function () {
        $workspace = createWorkspace();
        $service = app(AgentApiKeyService::class);

        $key = $service->create($workspace, 'One-time key');

        expect($key->plainTextKey)->not->toBeNull();

        $freshKey = AgentApiKey::find($key->id);
        expect($freshKey->plainTextKey)->toBeNull();
    });

    it('creates key with expiry date', function () {
        $workspace = createWorkspace();
        $service = app(AgentApiKeyService::class);
        $expiresAt = now()->addDays(30);

        $key = $service->create(
            workspace: $workspace,
            name: 'Expiring Key',
            expiresAt: $expiresAt,
        );

        expect($key->expires_at)->not->toBeNull()
            ->and($key->expires_at->toDateString())->toBe($expiresAt->toDateString());
    });

    it('creates key with no expiry when null passed', function () {
        $workspace = createWorkspace();
        $service = app(AgentApiKeyService::class);

        $key = $service->create($workspace, 'Permanent Key', expiresAt: null);

        expect($key->expires_at)->toBeNull();
    });
});

// =========================================================================
// Workspace Scoping (used by ApiKeyManager::revokeKey and render)
// =========================================================================

describe('ApiKeyManager workspace scoping', function () {
    it('forWorkspace scope returns only keys for given workspace', function () {
        $workspace1 = createWorkspace();
        $workspace2 = createWorkspace();

        $key1 = createApiKey($workspace1, 'Key for workspace 1');
        $key2 = createApiKey($workspace2, 'Key for workspace 2');

        $keys = AgentApiKey::forWorkspace($workspace1)->get();

        expect($keys)->toHaveCount(1)
            ->and($keys->first()->id)->toBe($key1->id);
    });

    it('forWorkspace accepts workspace model', function () {
        $workspace = createWorkspace();
        createApiKey($workspace, 'Key');

        $keys = AgentApiKey::forWorkspace($workspace)->get();

        expect($keys)->toHaveCount(1);
    });

    it('forWorkspace accepts workspace ID', function () {
        $workspace = createWorkspace();
        createApiKey($workspace, 'Key');

        $keys = AgentApiKey::forWorkspace($workspace->id)->get();

        expect($keys)->toHaveCount(1);
    });

    it('forWorkspace prevents cross-workspace key access', function () {
        $workspace1 = createWorkspace();
        $workspace2 = createWorkspace();

        $key = createApiKey($workspace1, 'Workspace 1 key');

        // Attempting to find workspace1's key while scoped to workspace2
        $found = AgentApiKey::forWorkspace($workspace2)->find($key->id);

        expect($found)->toBeNull();
    });
});

// =========================================================================
// Revoke Integration (as used by ApiKeyManager::revokeKey)
// =========================================================================

describe('ApiKeyManager key revocation integration', function () {
    it('revokes a key via service', function () {
        $workspace = createWorkspace();
        $key = createApiKey($workspace, 'Key to revoke');
        $service = app(AgentApiKeyService::class);

        expect($key->isActive())->toBeTrue();

        $service->revoke($key);

        expect($key->fresh()->isRevoked())->toBeTrue();
    });

    it('revoked key is inactive', function () {
        $workspace = createWorkspace();
        $key = createApiKey($workspace, 'Key to revoke');

        $key->revoke();

        expect($key->isActive())->toBeFalse()
            ->and($key->isRevoked())->toBeTrue();
    });

    it('revoking clears validation', function () {
        $workspace = createWorkspace();
        $key = createApiKey($workspace, 'Key to revoke');
        $service = app(AgentApiKeyService::class);

        $plainKey = $key->plainTextKey;
        $service->revoke($key);

        $validated = $service->validate($plainKey);
        expect($validated)->toBeNull();
    });
});

// =========================================================================
// Available Permissions (used by ApiKeyManager::availablePermissions)
// =========================================================================

describe('ApiKeyManager available permissions', function () {
    it('AgentApiKey provides available permissions list', function () {
        $permissions = AgentApiKey::availablePermissions();

        expect($permissions)
            ->toBeArray()
            ->toHaveKey(AgentApiKey::PERM_PLANS_READ)
            ->toHaveKey(AgentApiKey::PERM_PLANS_WRITE)
            ->toHaveKey(AgentApiKey::PERM_SESSIONS_READ)
            ->toHaveKey(AgentApiKey::PERM_SESSIONS_WRITE);
    });

    it('permission constants match available permissions keys', function () {
        $permissions = AgentApiKey::availablePermissions();

        expect(array_keys($permissions))
            ->toContain(AgentApiKey::PERM_PLANS_READ)
            ->toContain(AgentApiKey::PERM_PHASES_WRITE)
            ->toContain(AgentApiKey::PERM_TEMPLATES_READ);
    });

    it('key can be created with any available permission', function () {
        $workspace = createWorkspace();
        $allPermissions = array_keys(AgentApiKey::availablePermissions());

        $key = createApiKey($workspace, 'Full Access', $allPermissions);

        expect($key->permissions)->toBe($allPermissions);

        foreach ($allPermissions as $permission) {
            expect($key->hasPermission($permission))->toBeTrue();
        }
    });
});
