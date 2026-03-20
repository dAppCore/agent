<?php

declare(strict_types=1);

namespace Core\Mod\Agentic\Services\Concerns;

use Illuminate\Http\Client\ConnectionException;
use Illuminate\Http\Client\RequestException;
use Illuminate\Http\Client\Response;
use RuntimeException;

/**
 * Provides retry logic with exponential backoff for AI provider services.
 */
trait HasRetry
{
    protected int $maxRetries = 3;

    protected int $baseDelayMs = 1000;

    protected int $maxDelayMs = 30000;

    /**
     * Execute a callback with retry logic.
     *
     * @param  callable  $callback  Function that returns Response
     * @param  string  $provider  Provider name for error messages
     *
     * @throws RuntimeException
     */
    protected function withRetry(callable $callback, string $provider): Response
    {
        $lastException = null;

        for ($attempt = 1; $attempt <= $this->maxRetries; $attempt++) {
            try {
                $response = $callback();

                // Check for retryable HTTP status codes
                if ($response->successful()) {
                    return $response;
                }

                $status = $response->status();

                // Don't retry client errors (4xx) except rate limits
                if ($status >= 400 && $status < 500 && $status !== 429) {
                    throw new RuntimeException(
                        "{$provider} API error: ".$response->json('error.message', 'Request failed with status '.$status)
                    );
                }

                // Retryable: 429 (rate limit), 5xx (server errors)
                if ($status === 429 || $status >= 500) {
                    $lastException = new RuntimeException(
                        "{$provider} API error (attempt {$attempt}/{$this->maxRetries}): Status {$status}"
                    );

                    if ($attempt < $this->maxRetries) {
                        $this->sleep($this->calculateDelay($attempt, $response));
                    }

                    continue;
                }

                // Unexpected status code
                throw new RuntimeException(
                    "{$provider} API error: Unexpected status {$status}"
                );
            } catch (ConnectionException $e) {
                $lastException = new RuntimeException(
                    "{$provider} connection error (attempt {$attempt}/{$this->maxRetries}): ".$e->getMessage(),
                    0,
                    $e
                );

                if ($attempt < $this->maxRetries) {
                    $this->sleep($this->calculateDelay($attempt));
                }
            } catch (RequestException $e) {
                $lastException = new RuntimeException(
                    "{$provider} request error (attempt {$attempt}/{$this->maxRetries}): ".$e->getMessage(),
                    0,
                    $e
                );

                if ($attempt < $this->maxRetries) {
                    $this->sleep($this->calculateDelay($attempt));
                }
            }
        }

        throw $lastException ?? new RuntimeException("{$provider} API error: Unknown error after {$this->maxRetries} attempts");
    }

    /**
     * Calculate delay for next retry with exponential backoff and jitter.
     */
    protected function calculateDelay(int $attempt, ?Response $response = null): int
    {
        // Check for Retry-After header
        if ($response) {
            $retryAfter = $response->header('Retry-After');
            if ($retryAfter !== null) {
                // Retry-After can be seconds or HTTP-date
                if (is_numeric($retryAfter)) {
                    return min((int) $retryAfter * 1000, $this->maxDelayMs);
                }
            }
        }

        // Exponential backoff: base * 2^(attempt-1)
        $delay = $this->baseDelayMs * (2 ** ($attempt - 1));

        // Add jitter (0-25% of delay)
        $jitter = (int) ($delay * (mt_rand(0, 25) / 100));
        $delay += $jitter;

        return min($delay, $this->maxDelayMs);
    }

    /**
     * Sleep for the specified number of milliseconds.
     */
    protected function sleep(int $milliseconds): void
    {
        usleep($milliseconds * 1000);
    }
}
