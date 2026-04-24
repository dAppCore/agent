<?php

// SPDX-License-Identifier: EUPL-1.2

declare(strict_types=1);

use Core\Mcp\Services\CircuitBreaker;
use Core\Mod\Agentic\Mcp\Tools\Agent\Brain\BrainList;
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

test('CircuitBreaker_brain_list_Good_routes_failures_through_with_circuit_breaker', function (): void {
    $workspace = createWorkspace();
    $breaker = Mockery::mock(CircuitBreaker::class);

    $breaker->shouldReceive('call')
        ->once()
        ->with('brain', Mockery::type(Closure::class), Mockery::type(Closure::class))
        ->andReturnUsing(function (string $service, Closure $operation, ?Closure $fallback = null): array {
            return $fallback instanceof Closure ? $fallback() : [];
        });

    $this->app->instance(CircuitBreaker::class, $breaker);

    $result = (new BrainList)->handle([], [
        'workspace_id' => $workspace->id,
    ]);

    expect($result['code'])->toBe('service_unavailable')
        ->and($result['error'])->toBe('Brain service temporarily unavailable. Memory list unavailable.');
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
