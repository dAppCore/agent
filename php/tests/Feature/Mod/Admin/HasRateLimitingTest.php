<?php

// SPDX-License-Identifier: EUPL-1.2

declare(strict_types=1);

namespace Core\Mod\Agentic\Tests\Feature\Mod\Admin;

use Core\Mod\Agentic\Website\Hub\Concerns\HasRateLimiting;
use Illuminate\Support\Facades\RateLimiter;

class HasRateLimitingTest extends AdminTestCase
{
    public function test_HasRateLimiting_rateLimit_Good_executes_callback_before_limit(): void
    {
        $this->actingAsHades();

        $component = new class
        {
            use HasRateLimiting;

            public string $actionMessage = '';

            public string $actionType = '';

            public function attempt(): mixed
            {
                return $this->rateLimit('platform-test', 2, static fn (): string => 'ok');
            }
        };

        RateLimiter::clear('platform-test:1');

        $this->assertSame('ok', $component->attempt());
        $this->assertSame('ok', $component->attempt());
    }

    public function test_HasRateLimiting_rateLimit_Bad_sets_warning_state_when_limited(): void
    {
        $this->actingAsHades();

        $component = new class
        {
            use HasRateLimiting;

            public string $actionMessage = '';

            public string $actionType = '';

            public function attempt(): mixed
            {
                return $this->rateLimit('platform-test', 1, static fn (): string => 'ok');
            }
        };

        RateLimiter::clear('platform-test:1');
        $component->attempt();

        $this->assertNull($component->attempt());
        $this->assertSame('warning', $component->actionType);
        $this->assertStringContainsString('Too many platform test attempts', $component->actionMessage);
    }
}
