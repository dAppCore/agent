<?php

declare(strict_types=1);

/**
 * Tests for AgentToolRegistry caching behaviour (PERF-002).
 *
 * Verifies that forApiKey() caches results, that the cache is invalidated
 * when permissions change or a key is revoked, and that the TTL is honoured.
 */

use Core\Api\Models\ApiKey;
use Core\Mod\Agentic\Mcp\Tools\Agent\Contracts\AgentToolInterface;
use Core\Mod\Agentic\Services\AgentToolRegistry;
use Illuminate\Support\Facades\Cache;

// =========================================================================
// Helpers
// =========================================================================

/**
 * Build a minimal AgentToolInterface stub.
 */
function makeTool(string $name, array $scopes = [], string $category = 'test'): AgentToolInterface
{
    return new class($name, $scopes, $category) implements AgentToolInterface
    {
        public function __construct(
            private readonly string $toolName,
            private readonly array $toolScopes,
            private readonly string $toolCategory,
        ) {}

        public function name(): string
        {
            return $this->toolName;
        }

        public function description(): string
        {
            return 'Test tool';
        }

        public function inputSchema(): array
        {
            return [];
        }

        public function handle(array $args, array $context = []): array
        {
            return ['success' => true];
        }

        public function requiredScopes(): array
        {
            return $this->toolScopes;
        }

        public function category(): string
        {
            return $this->toolCategory;
        }
    };
}

/**
 * Build a minimal ApiKey mock with controllable scopes and tool_scopes.
 *
 * Uses Mockery to avoid requiring the real ApiKey class at load time,
 * since the php-api package is not available in this test environment.
 */
function makeApiKey(int $id, array $scopes = [], ?array $toolScopes = null, ?int $rateLimit = null): ApiKey
{
    $key = Mockery::mock(ApiKey::class);
    $key->shouldReceive('getKey')->andReturn($id);
    $key->shouldReceive('hasScope')->andReturnUsing(
        fn (string $scope) => in_array($scope, $scopes, true)
    );
    $key->tool_scopes = $toolScopes;
    if ($rateLimit !== null) {
        $key->rate_limit = $rateLimit;
    }

    return $key;
}

// =========================================================================
// Caching – basic behaviour
// =========================================================================

describe('forApiKey caching', function () {
    beforeEach(function () {
        Cache::flush();
    });

    it('returns the correct tools on first call (cache miss)', function () {
        $registry = new AgentToolRegistry;
        $registry->register(makeTool('plan.create', ['plans.write']));
        $registry->register(makeTool('session.start', ['sessions.write']));

        $apiKey = makeApiKey(1, ['plans.write', 'sessions.write']);

        $tools = $registry->forApiKey($apiKey);

        expect($tools->keys()->sort()->values()->all())
            ->toBe(['plan.create', 'session.start']);
    });

    it('stores permitted tool names in cache after first call', function () {
        $registry = new AgentToolRegistry;
        $registry->register(makeTool('plan.create', ['plans.write']));

        $apiKey = makeApiKey(42, ['plans.write']);

        $registry->forApiKey($apiKey);

        $cached = Cache::get('agent_tool_registry:api_key:42');
        expect($cached)->toBe(['plan.create']);
    });

    it('returns same result on second call (cache hit)', function () {
        $registry = new AgentToolRegistry;
        $registry->register(makeTool('plan.create', ['plans.write']));
        $registry->register(makeTool('session.start', ['sessions.write']));

        $apiKey = makeApiKey(1, ['plans.write']);

        $first = $registry->forApiKey($apiKey)->keys()->all();
        $second = $registry->forApiKey($apiKey)->keys()->all();

        expect($second)->toBe($first);
    });

    it('filters tools whose required scopes the key lacks', function () {
        $registry = new AgentToolRegistry;
        $registry->register(makeTool('plan.create', ['plans.write']));
        $registry->register(makeTool('session.start', ['sessions.write']));

        $apiKey = makeApiKey(1, ['plans.write']); // only plans.write

        $tools = $registry->forApiKey($apiKey);

        expect($tools->has('plan.create'))->toBeTrue()
            ->and($tools->has('session.start'))->toBeFalse();
    });

    it('respects tool_scopes allowlist on the api key', function () {
        $registry = new AgentToolRegistry;
        $registry->register(makeTool('plan.create', []));
        $registry->register(makeTool('session.start', []));

        $apiKey = makeApiKey(5, [], ['plan.create']); // explicitly restricted

        $tools = $registry->forApiKey($apiKey);

        expect($tools->has('plan.create'))->toBeTrue()
            ->and($tools->has('session.start'))->toBeFalse();
    });

    it('allows all tools when tool_scopes is null', function () {
        $registry = new AgentToolRegistry;
        $registry->register(makeTool('plan.create', []));
        $registry->register(makeTool('session.start', []));

        $apiKey = makeApiKey(7, [], null); // null = unrestricted

        $tools = $registry->forApiKey($apiKey);

        expect($tools)->toHaveCount(2);
    });

    it('caches separately per api key id', function () {
        $registry = new AgentToolRegistry;
        $registry->register(makeTool('plan.create', ['plans.write']));
        $registry->register(makeTool('session.start', ['sessions.write']));

        $keyA = makeApiKey(100, ['plans.write']);
        $keyB = makeApiKey(200, ['sessions.write']);

        $toolsA = $registry->forApiKey($keyA)->keys()->all();
        $toolsB = $registry->forApiKey($keyB)->keys()->all();

        expect($toolsA)->toBe(['plan.create'])
            ->and($toolsB)->toBe(['session.start']);

        expect(Cache::get('agent_tool_registry:api_key:100'))->toBe(['plan.create'])
            ->and(Cache::get('agent_tool_registry:api_key:200'))->toBe(['session.start']);
    });
});

// =========================================================================
// Cache TTL
// =========================================================================

describe('cache TTL', function () {
    it('declares CACHE_TTL constant as 3600 (1 hour)', function () {
        expect(AgentToolRegistry::CACHE_TTL)->toBe(3600);
    });

    it('stores entries in cache after first call', function () {
        Cache::flush();

        $registry = new AgentToolRegistry;
        $registry->register(makeTool('plan.create', []));

        $apiKey = makeApiKey(99, []);
        $registry->forApiKey($apiKey);

        expect(Cache::has('agent_tool_registry:api_key:99'))->toBeTrue();
    });
});

// =========================================================================
// Cache invalidation – flushCacheForApiKey
// =========================================================================

describe('flushCacheForApiKey', function () {
    beforeEach(function () {
        Cache::flush();
    });

    it('removes the cached entry for the given key id', function () {
        $registry = new AgentToolRegistry;
        $registry->register(makeTool('plan.create', []));

        $apiKey = makeApiKey(10, []);
        $registry->forApiKey($apiKey);

        expect(Cache::has('agent_tool_registry:api_key:10'))->toBeTrue();

        $registry->flushCacheForApiKey(10);

        expect(Cache::has('agent_tool_registry:api_key:10'))->toBeFalse();
    });

    it('re-fetches permitted tools after cache flush', function () {
        $registry = new AgentToolRegistry;
        $registry->register(makeTool('plan.create', []));

        $apiKey = makeApiKey(11, []);

        // Prime the cache (only plan.create at this point)
        expect($registry->forApiKey($apiKey)->keys()->all())->toBe(['plan.create']);

        $registry->flushCacheForApiKey(11);

        // Register an additional tool – should appear now that cache is gone
        $registry->register(makeTool('session.start', []));
        $after = $registry->forApiKey($apiKey)->keys()->sort()->values()->all();

        expect($after)->toBe(['plan.create', 'session.start']);
    });

    it('does not affect cache entries for other key ids', function () {
        $registry = new AgentToolRegistry;
        $registry->register(makeTool('plan.create', []));

        $key12 = makeApiKey(12, []);
        $key13 = makeApiKey(13, []);

        $registry->forApiKey($key12);
        $registry->forApiKey($key13);

        $registry->flushCacheForApiKey(12);

        expect(Cache::has('agent_tool_registry:api_key:12'))->toBeFalse()
            ->and(Cache::has('agent_tool_registry:api_key:13'))->toBeTrue();
    });

    it('accepts a string key id', function () {
        $registry = new AgentToolRegistry;
        $registry->register(makeTool('plan.create', []));

        $apiKey = makeApiKey(20, []);
        $registry->forApiKey($apiKey);

        $registry->flushCacheForApiKey('20');

        expect(Cache::has('agent_tool_registry:api_key:20'))->toBeFalse();
    });

    it('is a no-op when cache entry does not exist', function () {
        $registry = new AgentToolRegistry;

        // Should not throw when nothing is cached
        $registry->flushCacheForApiKey(999);

        expect(Cache::has('agent_tool_registry:api_key:999'))->toBeFalse();
    });
});

// =========================================================================
// Execution rate limiting
// =========================================================================

describe('execute rate limiting', function () {
    beforeEach(function () {
        Cache::flush();
    });

    it('records executions in a separate cache budget', function () {
        $registry = new AgentToolRegistry;
        $registry->register(makeTool('plan.create', ['plans.write']));

        $apiKey = makeApiKey(50, ['plans.write'], null, 2);

        $result = $registry->execute('plan.create', [], [], $apiKey, false);

        expect($result['success'])->toBeTrue()
            ->and(Cache::get('agent_api_key_tool_rate:50'))->toBe(1);
    });

    it('rejects executions once the budget is exhausted', function () {
        $registry = new AgentToolRegistry;
        $registry->register(makeTool('plan.create', ['plans.write']));

        $apiKey = makeApiKey(51, ['plans.write'], null, 1);
        Cache::put('agent_api_key_tool_rate:51', 1, 60);

        expect(fn () => $registry->execute('plan.create', [], [], $apiKey, false))
            ->toThrow(\RuntimeException::class, 'Rate limit exceeded');
    });
});
