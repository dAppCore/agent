<?php

// SPDX-License-Identifier: EUPL-1.2

declare(strict_types=1);

namespace Core\Mod\Agentic\Website\Hub\Concerns;

use Illuminate\Support\Facades\RateLimiter;

trait HasRateLimiting
{
    protected function rateLimit(
        string $action,
        int $maxAttempts,
        callable $callback,
        int $decaySeconds = 60
    ): mixed {
        $key = sprintf('%s:%s', $action, auth()->id() ?? 'guest');
        $executed = false;
        $result = null;

        RateLimiter::attempt($key, $maxAttempts, function () use (&$executed, &$result, $callback) {
            $executed = true;
            $result = $callback();
        }, $decaySeconds);

        if (! $executed) {
            $this->onRateLimited($action, $key);

            return null;
        }

        return $result;
    }

    protected function onRateLimited(string $action, string $key): void
    {
        $seconds = RateLimiter::availableIn($key);
        $message = sprintf('Too many %s attempts. Try again in %d seconds.', str_replace('-', ' ', $action), $seconds);

        if (property_exists($this, 'actionMessage')) {
            $this->actionMessage = $message;
        } else {
            session()->flash('warning', $message);
        }

        if (property_exists($this, 'actionType')) {
            $this->actionType = 'warning';
        }
    }
}
