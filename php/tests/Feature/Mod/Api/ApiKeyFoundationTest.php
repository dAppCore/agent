<?php

declare(strict_types=1);

use Core\Mod\Agentic\Mod\Api\Boot as ApiBoot;
use Core\Mod\Agentic\Mod\Api\Models\ApiKey;
use Illuminate\Support\Facades\DB;

beforeEach(function (): void {
    $this->app->register(ApiBoot::class);
});

describe('ApiKey foundation', function () {
    it('generates bcrypt-backed hk keys with the required format', function (): void {
        $workspace = createWorkspace();

        $result = ApiKey::generate($workspace->id, null, 'Gateway Key');

        expect($result['plain_key'])->toMatch('/^hk_[A-Za-z0-9]{8}_[A-Za-z0-9]{48}$/')
            ->and($result['api_key']->prefix)->toBe(substr($result['plain_key'], 0, 11))
            ->and(password_get_info($result['api_key']->key)['algoName'])->toBe('bcrypt')
            ->and(password_verify(explode('_', $result['plain_key'], 3)[2], $result['api_key']->key))->toBeTrue();
    });

    it('finds a key by prefix and candidate verification rather than hashing in the query', function (): void {
        $workspace = createWorkspace();
        $result = ApiKey::generate($workspace->id, null, 'Lookup Key');

        DB::flushQueryLog();
        DB::enableQueryLog();

        $found = ApiKey::findByPlainKey($result['plain_key']);
        $queries = collect(DB::getQueryLog())->pluck('query')->implode("\n");

        expect($found?->is($result['api_key']))->toBeTrue()
            ->and($queries)->toContain('prefix');
    });

    it('rejects malformed and expired keys', function (): void {
        $workspace = createWorkspace();
        $expired = ApiKey::generate($workspace->id, null, 'Expired Key', expiresAt: now()->subMinute());

        expect(ApiKey::findByPlainKey(''))->toBeNull()
            ->and(ApiKey::findByPlainKey('hk_short'))->toBeNull()
            ->and(ApiKey::findByPlainKey($expired['plain_key']))->toBeNull();
    });
});
