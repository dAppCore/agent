<?php

// SPDX-License-Identifier: EUPL-1.2

declare(strict_types=1);

use Core\Mcp\Services\CircuitBreaker;
use Core\Mod\Agentic\Mcp\Tools\Agent\Brain\BrainList;
use Core\Mod\Agentic\Models\BrainMemory;
use Core\Mod\Agentic\Services\BrainService;
use Illuminate\Http\Client\Request;
use Illuminate\Support\Facades\Http;

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

    Http::fake([
        'http://localhost:6334/collections/openbrain/points' => Http::sequence()
            ->push(['error' => 'unavailable'], 503)
            ->push(['result' => ['status' => 'ok']], 200),
    ]);

    $brain->qdrantUpsert([
        ['id' => 'memory-1', 'vector' => [0.1, 0.2], 'payload' => ['type' => 'fact']],
    ]);

    expect($brain->sleepCalls)->toBe([100]);

    Http::assertSentCount(2);
    Http::assertSent(fn (Request $request): bool => $request->url() === 'http://localhost:6334/collections/openbrain/points'
        && $request->method() === 'PUT');
});

test('CircuitBreaker_retryable_http_Ugly_does_not_retry_qdrant_requests_on_401', function (): void {
    $brain = retryableBrainService();

    Http::fake([
        'http://localhost:6334/collections/openbrain/points' => Http::sequence()
            ->push(['error' => 'unauthorised'], 401)
            ->push(['result' => ['status' => 'ok']], 200),
    ]);

    expect(fn () => $brain->qdrantUpsert([
        ['id' => 'memory-2', 'vector' => [0.3, 0.4], 'payload' => ['type' => 'fact']],
    ]))->toThrow(RuntimeException::class, 'Qdrant upsert failed: 401');

    expect($brain->sleepCalls)->toBe([]);
    Http::assertSentCount(1);
});

test('CircuitBreaker_retryable_http_retries_qdrant_requests_on_429_using_retry_after_header', function (): void {
    $brain = retryableBrainService();

    Http::fake([
        'http://localhost:6334/collections/openbrain/points' => Http::sequence()
            ->push(['error' => 'rate limited'], 429, ['Retry-After' => '2'])
            ->push(['result' => ['status' => 'ok']], 200),
    ]);

    $brain->qdrantUpsert([
        ['id' => 'memory-3', 'vector' => [0.5, 0.6], 'payload' => ['type' => 'fact']],
    ]);

    expect($brain->sleepCalls)->toBe([2000]);

    Http::assertSentCount(2);
});

test('CircuitBreaker_retryable_http_retries_qdrant_requests_on_408', function (): void {
    $brain = retryableBrainService();

    Http::fake([
        'http://localhost:6334/collections/openbrain/points' => Http::sequence()
            ->push(['error' => 'timeout'], 408)
            ->push(['result' => ['status' => 'ok']], 200),
    ]);

    $brain->qdrantUpsert([
        ['id' => 'memory-4', 'vector' => [0.7, 0.8], 'payload' => ['type' => 'fact']],
    ]);

    expect($brain->sleepCalls)->toBe([100]);

    Http::assertSentCount(2);
});
