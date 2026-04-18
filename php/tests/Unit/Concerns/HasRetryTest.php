<?php

declare(strict_types=1);

/**
 * Tests for the HasRetry trait.
 *
 * Exercises retry logic, exponential backoff, and error classification
 * in isolation from any real HTTP provider.
 */

use Core\Mod\Agentic\Services\Concerns\HasRetry;
use GuzzleHttp\Psr7\Response as PsrResponse;
use Illuminate\Http\Client\ConnectionException;
use Illuminate\Http\Client\RequestException;
use Illuminate\Http\Client\Response;

// ---------------------------------------------------------------------------
// Test helpers
// ---------------------------------------------------------------------------

/**
 * Build a testable object that uses the HasRetry trait.
 *
 * sleep() is overridden so tests run without actual delays.
 * The recorded sleep durations are accessible via ->sleepCalls.
 */
function retryService(int $maxRetries = 3, int $baseDelayMs = 1000, int $maxDelayMs = 30000): object
{
    return new class($maxRetries, $baseDelayMs, $maxDelayMs)
    {
        use HasRetry;

        public array $sleepCalls = [];

        public function __construct(int $maxRetries, int $baseDelayMs, int $maxDelayMs)
        {
            $this->maxRetries = $maxRetries;
            $this->baseDelayMs = $baseDelayMs;
            $this->maxDelayMs = $maxDelayMs;
        }

        public function runWithRetry(callable $callback, string $provider): Response
        {
            return $this->withRetry($callback, $provider);
        }

        public function computeDelay(int $attempt, ?Response $response = null): int
        {
            return $this->calculateDelay($attempt, $response);
        }

        protected function sleep(int $milliseconds): void
        {
            $this->sleepCalls[] = $milliseconds;
        }
    };
}

/**
 * Build an Illuminate Response wrapping a real PSR-7 response.
 *
 * @param  array<string,string>  $headers
 */
function fakeHttpResponse(int $status, array $body = [], array $headers = []): Response
{
    return new Response(new PsrResponse($status, $headers, json_encode($body)));
}

// ---------------------------------------------------------------------------
// withRetry – success paths
// ---------------------------------------------------------------------------

describe('withRetry success', function () {
    it('returns response immediately on first-attempt success', function () {
        $service = retryService();
        $response = fakeHttpResponse(200, ['ok' => true]);

        $result = $service->runWithRetry(fn () => $response, 'TestProvider');

        expect($result->successful())->toBeTrue();
        expect($service->sleepCalls)->toBeEmpty();
    });

    it('returns response after one transient 429 failure', function () {
        $service = retryService();
        $calls = 0;

        $result = $service->runWithRetry(function () use (&$calls) {
            $calls++;

            return $calls === 1
                ? fakeHttpResponse(429)
                : fakeHttpResponse(200, ['ok' => true]);
        }, 'TestProvider');

        expect($result->successful())->toBeTrue();
        expect($calls)->toBe(2);
    });

    it('returns response after one transient 500 failure', function () {
        $service = retryService();
        $calls = 0;

        $result = $service->runWithRetry(function () use (&$calls) {
            $calls++;

            return $calls === 1
                ? fakeHttpResponse(500)
                : fakeHttpResponse(200, ['ok' => true]);
        }, 'TestProvider');

        expect($result->successful())->toBeTrue();
        expect($calls)->toBe(2);
    });

    it('returns response after one ConnectionException', function () {
        $service = retryService();
        $calls = 0;

        $result = $service->runWithRetry(function () use (&$calls) {
            $calls++;
            if ($calls === 1) {
                throw new ConnectionException('Network error');
            }

            return fakeHttpResponse(200, ['ok' => true]);
        }, 'TestProvider');

        expect($result->successful())->toBeTrue();
        expect($calls)->toBe(2);
    });

    it('returns response after one RequestException', function () {
        $service = retryService();
        $calls = 0;

        $result = $service->runWithRetry(function () use (&$calls) {
            $calls++;
            if ($calls === 1) {
                throw new RequestException(fakeHttpResponse(503));
            }

            return fakeHttpResponse(200, ['ok' => true]);
        }, 'TestProvider');

        expect($result->successful())->toBeTrue();
        expect($calls)->toBe(2);
    });
});

// ---------------------------------------------------------------------------
// withRetry – max retry limits
// ---------------------------------------------------------------------------

describe('withRetry max retry limits', function () {
    it('throws after exhausting all retries on persistent 429', function () {
        $service = retryService(maxRetries: 3);
        $calls = 0;

        expect(function () use ($service, &$calls) {
            $service->runWithRetry(function () use (&$calls) {
                $calls++;

                return fakeHttpResponse(429);
            }, 'TestProvider');
        })->toThrow(RuntimeException::class);

        expect($calls)->toBe(3);
    });

    it('throws after exhausting all retries on persistent 500', function () {
        $service = retryService(maxRetries: 3);
        $calls = 0;

        expect(function () use ($service, &$calls) {
            $service->runWithRetry(function () use (&$calls) {
                $calls++;

                return fakeHttpResponse(500);
            }, 'TestProvider');
        })->toThrow(RuntimeException::class);

        expect($calls)->toBe(3);
    });

    it('throws after exhausting all retries on persistent ConnectionException', function () {
        $service = retryService(maxRetries: 2);
        $calls = 0;

        expect(function () use ($service, &$calls) {
            $service->runWithRetry(function () use (&$calls) {
                $calls++;
                throw new ConnectionException('Timeout');
            }, 'TestProvider');
        })->toThrow(RuntimeException::class, 'connection error');

        expect($calls)->toBe(2);
    });

    it('respects a custom maxRetries of 1 (no retries)', function () {
        $service = retryService(maxRetries: 1);
        $calls = 0;

        expect(function () use ($service, &$calls) {
            $service->runWithRetry(function () use (&$calls) {
                $calls++;

                return fakeHttpResponse(500);
            }, 'TestProvider');
        })->toThrow(RuntimeException::class);

        expect($calls)->toBe(1);
    });

    it('error message includes provider name', function () {
        $service = retryService(maxRetries: 1);

        expect(fn () => $service->runWithRetry(fn () => fakeHttpResponse(500), 'MyProvider'))
            ->toThrow(RuntimeException::class, 'MyProvider');
    });
});

// ---------------------------------------------------------------------------
// withRetry – non-retryable errors
// ---------------------------------------------------------------------------

describe('withRetry non-retryable client errors', function () {
    it('throws immediately on 401 without retrying', function () {
        $service = retryService(maxRetries: 3);
        $calls = 0;

        expect(function () use ($service, &$calls) {
            $service->runWithRetry(function () use (&$calls) {
                $calls++;

                return fakeHttpResponse(401, ['error' => ['message' => 'Unauthorised']]);
            }, 'TestProvider');
        })->toThrow(RuntimeException::class, 'TestProvider API error');

        expect($calls)->toBe(1);
    });

    it('throws immediately on 400 without retrying', function () {
        $service = retryService(maxRetries: 3);
        $calls = 0;

        expect(function () use ($service, &$calls) {
            $service->runWithRetry(function () use (&$calls) {
                $calls++;

                return fakeHttpResponse(400);
            }, 'TestProvider');
        })->toThrow(RuntimeException::class);

        expect($calls)->toBe(1);
    });

    it('throws immediately on 404 without retrying', function () {
        $service = retryService(maxRetries: 3);
        $calls = 0;

        expect(function () use ($service, &$calls) {
            $service->runWithRetry(function () use (&$calls) {
                $calls++;

                return fakeHttpResponse(404);
            }, 'TestProvider');
        })->toThrow(RuntimeException::class);

        expect($calls)->toBe(1);
    });
});

// ---------------------------------------------------------------------------
// withRetry – sleep (backoff) behaviour
// ---------------------------------------------------------------------------

describe('withRetry exponential backoff', function () {
    it('sleeps between retries but not after the final attempt', function () {
        $service = retryService(maxRetries: 3, baseDelayMs: 100, maxDelayMs: 10000);

        try {
            $service->runWithRetry(fn () => fakeHttpResponse(500), 'TestProvider');
        } catch (RuntimeException) {
            // expected
        }

        // 3 attempts → 2 sleeps (between attempt 1-2 and 2-3)
        expect($service->sleepCalls)->toHaveCount(2);
    });

    it('does not sleep when succeeding on first attempt', function () {
        $service = retryService();

        $service->runWithRetry(fn () => fakeHttpResponse(200), 'TestProvider');

        expect($service->sleepCalls)->toBeEmpty();
    });

    it('sleeps once when succeeding on the second attempt', function () {
        $service = retryService(maxRetries: 3, baseDelayMs: 100, maxDelayMs: 10000);
        $calls = 0;

        $service->runWithRetry(function () use (&$calls) {
            $calls++;

            return $calls === 1 ? fakeHttpResponse(500) : fakeHttpResponse(200);
        }, 'TestProvider');

        expect($service->sleepCalls)->toHaveCount(1);
        expect($service->sleepCalls[0])->toBeGreaterThanOrEqual(100);
    });
});

// ---------------------------------------------------------------------------
// calculateDelay – exponential backoff formula
// ---------------------------------------------------------------------------

describe('calculateDelay', function () {
    it('returns base delay for attempt 1', function () {
        $service = retryService(baseDelayMs: 1000, maxDelayMs: 30000);

        // delay = 1000 * 2^0 = 1000ms, plus up to 25% jitter
        $delay = $service->computeDelay(1);

        expect($delay)->toBeGreaterThanOrEqual(1000)
            ->and($delay)->toBeLessThanOrEqual(1250);
    });

    it('doubles the delay for attempt 2', function () {
        $service = retryService(baseDelayMs: 1000, maxDelayMs: 30000);

        // delay = 1000 * 2^1 = 2000ms, plus up to 25% jitter
        $delay = $service->computeDelay(2);

        expect($delay)->toBeGreaterThanOrEqual(2000)
            ->and($delay)->toBeLessThanOrEqual(2500);
    });

    it('quadruples the delay for attempt 3', function () {
        $service = retryService(baseDelayMs: 1000, maxDelayMs: 30000);

        // delay = 1000 * 2^2 = 4000ms, plus up to 25% jitter
        $delay = $service->computeDelay(3);

        expect($delay)->toBeGreaterThanOrEqual(4000)
            ->and($delay)->toBeLessThanOrEqual(5000);
    });

    it('caps the delay at maxDelayMs', function () {
        $service = retryService(baseDelayMs: 10000, maxDelayMs: 5000);

        // 10000 * 2^0 = 10000ms → capped at 5000ms
        $delay = $service->computeDelay(1);

        expect($delay)->toBe(5000);
    });

    it('respects numeric Retry-After header (in seconds)', function () {
        $service = retryService(maxDelayMs: 60000);
        $response = fakeHttpResponse(429, [], ['Retry-After' => '10']);

        // Retry-After is 10 seconds = 10000ms
        $delay = $service->computeDelay(1, $response);

        expect($delay)->toBe(10000);
    });

    it('caps Retry-After header value at maxDelayMs', function () {
        $service = retryService(maxDelayMs: 5000);
        $response = fakeHttpResponse(429, [], ['Retry-After' => '60']);

        // 60 seconds = 60000ms → capped at 5000ms
        $delay = $service->computeDelay(1, $response);

        expect($delay)->toBe(5000);
    });

    it('falls back to exponential backoff when no Retry-After header', function () {
        $service = retryService(baseDelayMs: 1000, maxDelayMs: 30000);
        $response = fakeHttpResponse(500);

        $delay = $service->computeDelay(1, $response);

        expect($delay)->toBeGreaterThanOrEqual(1000)
            ->and($delay)->toBeLessThanOrEqual(1250);
    });
});
