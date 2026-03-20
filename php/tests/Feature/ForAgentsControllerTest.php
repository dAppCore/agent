<?php

declare(strict_types=1);

/**
 * Tests for ForAgentsController cache key namespacing (CQ-003).
 *
 * Verifies that the cache key is config-based to prevent cross-module collisions,
 * and that cache invalidation uses the same namespaced key.
 */

use Core\Mod\Agentic\Controllers\ForAgentsController;
use Illuminate\Support\Facades\Cache;

// =========================================================================
// Cache Key Tests
// =========================================================================

describe('ForAgentsController cache key', function () {
    it('uses the default namespaced cache key', function () {
        $controller = new ForAgentsController;

        expect($controller->cacheKey())->toBe('agentic.for-agents.json');
    });

    it('uses a custom cache key when configured', function () {
        config(['mcp.cache.for_agents_key' => 'custom-module.for-agents.json']);

        $controller = new ForAgentsController;

        expect($controller->cacheKey())->toBe('custom-module.for-agents.json');
    });

    it('returns to default key after config is cleared', function () {
        config(['mcp.cache.for_agents_key' => null]);

        $controller = new ForAgentsController;

        expect($controller->cacheKey())->toBe('agentic.for-agents.json');
    });
});

// =========================================================================
// Cache Behaviour Tests
// =========================================================================

describe('ForAgentsController cache behaviour', function () {
    it('stores data under the namespaced cache key', function () {
        Cache::fake();

        $controller = new ForAgentsController;
        $controller();

        $key = $controller->cacheKey();
        expect(Cache::has($key))->toBeTrue();
    });

    it('returns cached data on subsequent calls', function () {
        Cache::fake();

        $controller = new ForAgentsController;
        $first = $controller();
        $second = $controller();

        expect($first->getContent())->toBe($second->getContent());
    });

    it('respects the configured TTL', function () {
        config(['mcp.cache.for_agents_ttl' => 7200]);
        Cache::fake();

        $controller = new ForAgentsController;
        $response = $controller();

        expect($response->headers->get('Cache-Control'))->toContain('max-age=7200');
    });

    it('uses default TTL of 3600 when not configured', function () {
        config(['mcp.cache.for_agents_ttl' => null]);
        Cache::fake();

        $controller = new ForAgentsController;
        $response = $controller();

        expect($response->headers->get('Cache-Control'))->toContain('max-age=3600');
    });

    it('can be invalidated using the namespaced key', function () {
        Cache::fake();

        $controller = new ForAgentsController;
        $controller();

        $key = $controller->cacheKey();
        expect(Cache::has($key))->toBeTrue();

        Cache::forget($key);
        expect(Cache::has($key))->toBeFalse();
    });

    it('stores data under the custom key when configured', function () {
        config(['mcp.cache.for_agents_key' => 'tenant-a.for-agents.json']);
        Cache::fake();

        $controller = new ForAgentsController;
        $controller();

        expect(Cache::has('tenant-a.for-agents.json'))->toBeTrue();
        expect(Cache::has('agentic.for-agents.json'))->toBeFalse();
    });
});

// =========================================================================
// Response Structure Tests
// =========================================================================

describe('ForAgentsController response', function () {
    it('returns a JSON response', function () {
        Cache::fake();

        $controller = new ForAgentsController;
        $response = $controller();

        expect($response->headers->get('Content-Type'))->toContain('application/json');
    });

    it('response contains platform information', function () {
        Cache::fake();

        $controller = new ForAgentsController;
        $response = $controller();
        $data = json_decode($response->getContent(), true);

        expect($data)->toHaveKey('platform')
            ->and($data['platform'])->toHaveKey('name');
    });

    it('response contains capabilities', function () {
        Cache::fake();

        $controller = new ForAgentsController;
        $response = $controller();
        $data = json_decode($response->getContent(), true);

        expect($data)->toHaveKey('capabilities')
            ->and($data['capabilities'])->toHaveKey('mcp_servers');
    });
});
