<?php

// SPDX-License-Identifier: EUPL-1.2

declare(strict_types=1);

use Core\Mcp\Services\CircuitBreaker;
use Core\Mod\Agentic\Mcp\Tools\Agent\Brain\BrainList;
use Core\Mod\Agentic\Models\BrainMemory;
use Core\Mod\Agentic\Services\BrainService;
use Illuminate\Http\Client\Request;
use Illuminate\Support\Facades\Cache;
use Illuminate\Support\Facades\Http;

beforeEach(function (): void {
    Cache::flush();
    config()->set('mcp.circuit_breaker.default_threshold', 5);
    config()->set('mcp.circuit_breaker.default_reset_timeout', 60);
    config()->set('mcp.circuit_breaker.default_failure_window', 120);
});

function retryableBrainService(): BrainService
{
    return new class extends BrainService
    {
        public array $sleepCalls = [];

        protected function sleepMilliseconds(int $milliseconds): void
        {
            $this->sleepCalls[] = $milliseconds;
        }
    };
}

/**
 * @return array{memory: BrainMemory, point: array{id: string, vector: array<float>, payload: array<string, string>}}
 */
function retryableQdrantFixture(array $attributes = []): array
{
    $workspace = createWorkspace();
    $memory = BrainMemory::create(array_merge([
        'workspace_id' => $workspace->id,
        'agent_id' => 'virgil',
        'type' => 'fact',
        'content' => 'Brain retry fixture.',
        'confidence' => 0.8,
    ], $attributes));

    return [
        'memory' => $memory,
        'point' => [
            'id' => $memory->id,
            'vector' => [0.1, 0.2],
            'payload' => ['type' => $memory->type],
        ],
    ];
}

test('CircuitBreaker_brain_list_Good_executes_without_circuit_breaker_for_db_only_query', function (): void {
    $workspace = createWorkspace();
    $breaker = Mockery::mock(CircuitBreaker::class);
    BrainMemory::create([
        'workspace_id' => $workspace->id,
        'agent_id' => 'virgil',
        'type' => 'fact',
        'content' => 'DB-backed list result.',
        'confidence' => 0.8,
    ]);

    $breaker->shouldNotReceive('call');

    $this->app->instance(CircuitBreaker::class, $breaker);

    $result = (new BrainList)->handle([], [
        'workspace_id' => $workspace->id,
    ]);

    expect($result['success'])->toBeTrue()
        ->and($result['count'])->toBe(1)
        ->and($result['memories'][0]['content'])->toBe('DB-backed list result.');
});

test('CircuitBreaker_retryable_http_Bad_retries_qdrant_requests_on_503', function (): void {
    $brain = retryableBrainService();
    $fixture = retryableQdrantFixture();

    Http::fake([
        'http://localhost:6334/collections/openbrain/points' => Http::sequence()
            ->push(['error' => 'unavailable'], 503)
            ->push(['result' => ['status' => 'ok']], 200),
    ]);

    $brain->qdrantUpsert([$fixture['point']]);

    expect($brain->sleepCalls)->toBe([100]);

    Http::assertSentCount(2);
    Http::assertSent(fn (Request $request): bool => $request->url() === 'http://localhost:6334/collections/openbrain/points'
        && $request->method() === 'PUT');
});

test('CircuitBreaker_retryable_http_Ugly_does_not_retry_qdrant_requests_on_401', function (): void {
    $brain = retryableBrainService();
    $fixture = retryableQdrantFixture();

    Http::fake([
        'http://localhost:6334/collections/openbrain/points' => Http::sequence()
            ->push(['error' => 'unauthorised'], 401)
            ->push(['result' => ['status' => 'ok']], 200),
    ]);

    expect(fn () => $brain->qdrantUpsert([$fixture['point']]))
        ->toThrow(RuntimeException::class, 'Qdrant upsert failed: 401');

    expect($brain->sleepCalls)->toBe([]);
    Http::assertSentCount(1);
});

test('CircuitBreaker_retryable_http_retries_qdrant_requests_on_429_using_retry_after_header', function (): void {
    $brain = retryableBrainService();
    $fixture = retryableQdrantFixture();

    Http::fake([
        'http://localhost:6334/collections/openbrain/points' => Http::sequence()
            ->push(['error' => 'rate limited'], 429, ['Retry-After' => '2'])
            ->push(['result' => ['status' => 'ok']], 200),
    ]);

    $brain->qdrantUpsert([$fixture['point']]);

    expect($brain->sleepCalls)->toBe([2000]);

    Http::assertSentCount(2);
});

test('CircuitBreaker_retryable_http_retries_qdrant_requests_on_408', function (): void {
    $brain = retryableBrainService();
    $fixture = retryableQdrantFixture();

    Http::fake([
        'http://localhost:6334/collections/openbrain/points' => Http::sequence()
            ->push(['error' => 'timeout'], 408)
            ->push(['result' => ['status' => 'ok']], 200),
    ]);

    $brain->qdrantUpsert([$fixture['point']]);

    expect($brain->sleepCalls)->toBe([100]);

    Http::assertSentCount(2);
});

test('CircuitBreaker_retryable_http_Good_recovers_from_five_transient_503s_without_opening_the_circuit', function (): void {
    $brain = retryableBrainService();
    $breaker = new CircuitBreaker;
    $fixture = retryableQdrantFixture();
    $sequence = Http::sequence();

    for ($attempt = 0; $attempt < 5; $attempt++) {
        $sequence->push(['error' => 'unavailable'], 503);
    }

    $sequence->push(['result' => ['status' => 'ok']], 200);

    Http::fake([
        'http://localhost:6334/collections/openbrain/points' => $sequence,
    ]);

    $breaker->call('brain', fn (): mixed => $brain->qdrantUpsert([$fixture['point']]));

    expect($brain->sleepCalls)->toHaveCount(5)
        ->and($breaker->getState('brain'))->toBe(CircuitBreaker::STATE_CLOSED)
        ->and($breaker->getStats('brain')['failures'])->toBe(0)
        ->and($breaker->getStats('brain')['successes'])->toBe(1);

    Http::assertSentCount(6);
});

test('CircuitBreaker_retryable_http_Bad_counts_exhausted_503_retries_as_one_failure', function (): void {
    $brain = retryableBrainService();
    $breaker = new CircuitBreaker;
    $fixture = retryableQdrantFixture();
    $sequence = Http::sequence();

    for ($attempt = 0; $attempt < 6; $attempt++) {
        $sequence->push(['error' => 'unavailable'], 503);
    }

    Http::fake([
        'http://localhost:6334/collections/openbrain/points' => $sequence,
    ]);

    try {
        $breaker->call('brain', fn (): mixed => $brain->qdrantUpsert([$fixture['point']]));
        $this->fail('Expected Qdrant failure after exhausting retries.');
    } catch (RuntimeException $exception) {
        expect($exception->getMessage())->toBe('Qdrant upsert failed: 503');
    }

    expect($brain->sleepCalls)->toHaveCount(5)
        ->and($breaker->getState('brain'))->toBe(CircuitBreaker::STATE_CLOSED)
        ->and($breaker->getStats('brain')['failures'])->toBe(1)
        ->and($breaker->getStats('brain')['successes'])->toBe(0);

    Http::assertSentCount(6);
});

test('CircuitBreaker_retryable_http_Good_honours_retry_after_without_counting_a_failure', function (): void {
    $brain = retryableBrainService();
    $breaker = new CircuitBreaker;
    $fixture = retryableQdrantFixture();

    Http::fake([
        'http://localhost:6334/collections/openbrain/points' => Http::sequence()
            ->push(['error' => 'rate limited'], 429, ['Retry-After' => '1'])
            ->push(['result' => ['status' => 'ok']], 200),
    ]);

    $breaker->call('brain', fn (): mixed => $brain->qdrantUpsert([$fixture['point']]));

    expect($brain->sleepCalls)->toBe([1000])
        ->and($breaker->getState('brain'))->toBe(CircuitBreaker::STATE_CLOSED)
        ->and($breaker->getStats('brain')['failures'])->toBe(0)
        ->and($breaker->getStats('brain')['successes'])->toBe(1);

    Http::assertSentCount(2);
});

test('CircuitBreaker_retryable_http_Ugly_treats_500_as_an_immediate_circuit_failure', function (): void {
    $brain = retryableBrainService();
    $breaker = new CircuitBreaker;
    $fixture = retryableQdrantFixture();

    Http::fake([
        'http://localhost:6334/collections/openbrain/points' => Http::sequence()
            ->push(['error' => 'server error'], 500)
            ->push(['result' => ['status' => 'ok']], 200),
    ]);

    try {
        $breaker->call('brain', fn (): mixed => $brain->qdrantUpsert([$fixture['point']]));
        $this->fail('Expected non-retryable 500 response to fail immediately.');
    } catch (RuntimeException $exception) {
        expect($exception->getMessage())->toBe('Qdrant upsert failed: 500');
    }

    expect($brain->sleepCalls)->toBe([])
        ->and($breaker->getState('brain'))->toBe(CircuitBreaker::STATE_CLOSED)
        ->and($breaker->getStats('brain')['failures'])->toBe(1)
        ->and($breaker->getStats('brain')['successes'])->toBe(0);

    Http::assertSentCount(1);
});
