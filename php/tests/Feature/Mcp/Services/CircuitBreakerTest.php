<?php

// SPDX-License-Identifier: EUPL-1.2

declare(strict_types=1);

require_once dirname(__DIR__).'/Support/bootstrap.php';

mcpRequire('Mcp/Services/CircuitBreaker.php');

use Core\Mcp\Exceptions\CircuitOpenException;
use Core\Mcp\Services\CircuitBreaker;
use Illuminate\Support\Facades\Cache;

beforeEach(function (): void {
    Cache::flush();
    config()->set('mcp.circuit_breaker.default_threshold', 2);
    config()->set('mcp.circuit_breaker.default_reset_timeout', 60);
    config()->set('mcp.circuit_breaker.default_failure_window', 120);
});

test('CircuitBreaker_call_Good_allows_closed_operations_and_records_success', function (): void {
    $breaker = new CircuitBreaker;

    $result = $breaker->call('agentic', static fn (): string => 'ok');

    expect($result)->toBe('ok')
        ->and($breaker->getState('agentic'))->toBe(CircuitBreaker::STATE_CLOSED)
        ->and($breaker->getStats('agentic')['successes'])->toBe(1);
});

test('CircuitBreaker_call_Bad_trips_open_and_fails_fast_without_a_fallback', function (): void {
    $breaker = new CircuitBreaker;

    try {
        $breaker->call('agentic', static function (): never {
            throw new RuntimeException('Connection refused');
        });
    } catch (RuntimeException) {
    }

    try {
        $breaker->call('agentic', static function (): never {
            throw new RuntimeException('Connection refused');
        });
    } catch (RuntimeException) {
    }

    expect($breaker->getState('agentic'))->toBe(CircuitBreaker::STATE_OPEN);

    $breaker->call('agentic', static fn (): string => 'ignored');
})->throws(CircuitOpenException::class);

test('CircuitBreaker_call_Ugly_uses_fallback_when_half_open_trial_is_already_locked', function (): void {
    $breaker = new CircuitBreaker;

    Cache::put('circuit_breaker:content:state', CircuitBreaker::STATE_OPEN, 86400);
    Cache::put('circuit_breaker:content:opened_at', time() - 120, 86400);
    Cache::put('circuit_breaker:content:trial_lock', true, 30);

    $result = $breaker->call(
        'content',
        static fn (): string => 'never',
        static fn (): string => 'fallback',
    );

    expect($result)->toBe('fallback')
        ->and($breaker->getState('content'))->toBe(CircuitBreaker::STATE_HALF_OPEN);
});
