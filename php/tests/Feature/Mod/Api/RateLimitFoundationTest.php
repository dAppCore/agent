<?php

declare(strict_types=1);

use Carbon\Carbon;
use Core\Mod\Agentic\Mod\Api\RateLimit\RateLimit;
use Core\Mod\Agentic\Mod\Api\RateLimit\RateLimitService;
use Illuminate\Cache\ArrayStore;
use Illuminate\Cache\Repository;

beforeEach(function (): void {
    Carbon::setTestNow(Carbon::parse('2026-04-25 12:00:00'));

    $this->service = new RateLimitService(new Repository(new ArrayStore));
});

afterEach(function (): void {
    Carbon::setTestNow();
});

describe('RateLimit foundation', function () {
    it('tracks hits separately from limit checks using a sliding window', function (): void {
        $rateLimit = new RateLimit(limit: 2, window: 60);
        $bucket = $this->service->buildEndpointKey('api_key:demo', 'docs.index');

        $checked = $this->service->checkLimit('demo', 'docs.index', $rateLimit);
        $recorded = $this->service->recordHit('demo', 'docs.index', $rateLimit);

        expect($checked->allowed)->toBeTrue()
            ->and($checked->remaining)->toBe(2)
            ->and($recorded->allowed)->toBeTrue()
            ->and($recorded->remaining)->toBe(1)
            ->and($this->service->attempts($bucket, 60))->toBe(1);
    });

    it('denies requests once the current window is full', function (): void {
        $rateLimit = new RateLimit(limit: 2, window: 60);

        $this->service->recordHit('demo', 'docs.index', $rateLimit);
        $this->service->recordHit('demo', 'docs.index', $rateLimit);

        $result = $this->service->checkLimit('demo', 'docs.index', $rateLimit);

        expect($result->allowed)->toBeFalse()
            ->and($result->remaining)->toBe(0)
            ->and($result->retryAfter)->toBe(60);
    });

    it('expires old hits and rejects invalid limit definitions', function (): void {
        $rateLimit = new RateLimit(limit: 2, window: 60);

        $this->service->recordHit('demo', 'docs.index', $rateLimit);
        Carbon::setTestNow(Carbon::parse('2026-04-25 12:01:01'));

        expect($this->service->checkLimit('demo', 'docs.index', $rateLimit)->allowed)->toBeTrue();
        expect(fn () => $this->service->checkLimit('demo', 'docs.index', new RateLimit(limit: 0, window: 60)))
            ->toThrow(InvalidArgumentException::class);
    });
});
