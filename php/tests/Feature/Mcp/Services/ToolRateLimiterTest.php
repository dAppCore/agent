<?php

// SPDX-License-Identifier: EUPL-1.2

declare(strict_types=1);

require_once dirname(__DIR__).'/Support/bootstrap.php';

mcpRequire('Mcp/Services/ToolRateLimiter.php');

use Core\Mcp\Services\ToolRateLimiter;
use Illuminate\Support\Facades\Cache;

beforeEach(function (): void {
    Cache::flush();
    config()->set('mcp.rate_limiting.enabled', true);
    config()->set('mcp.rate_limiting.decay_seconds', 60);
    config()->set('mcp.rate_limiting.calls_per_minute', 2);
    config()->set('mcp.rate_limiting.per_tool', ['send_email' => 1]);
});

test('ToolRateLimiter_check_Good_reports_remaining_calls_before_the_limit_is_hit', function (): void {
    $limiter = new ToolRateLimiter;

    $status = $limiter->check('sess-1', 'list_posts');
    $limiter->hit('sess-1', 'list_posts');
    $afterHit = $limiter->getStatus('sess-1', 'list_posts');

    expect($status['limited'])->toBeFalse()
        ->and($status['remaining'])->toBe(1)
        ->and($afterHit['remaining'])->toBe(1);
});

test('ToolRateLimiter_check_Bad_applies_tool_specific_limits_and_returns_retry_after_when_limited', function (): void {
    $limiter = new ToolRateLimiter;

    $limiter->hit('sess-2', 'send_email');
    $result = $limiter->check('sess-2', 'send_email');

    expect($result['limited'])->toBeTrue()
        ->and($result['remaining'])->toBe(0)
        ->and($result['retry_after'])->toBeInt();
});

test('ToolRateLimiter_hit_Ugly_uses_put_for_the_first_call_and_increment_for_subsequent_calls', function (): void {
    Cache::shouldReceive('get')->once()->with('mcp_rate_limit:sess-3:list_posts', 0)->andReturn(0);
    Cache::shouldReceive('put')->once()->with('mcp_rate_limit:sess-3:list_posts', 1, 60);
    Cache::shouldReceive('get')->once()->with('mcp_rate_limit:sess-3:list_posts', 0)->andReturn(1);
    Cache::shouldReceive('increment')->once()->with('mcp_rate_limit:sess-3:list_posts');

    $limiter = new ToolRateLimiter;
    $limiter->hit('sess-3', 'list_posts');
    $limiter->hit('sess-3', 'list_posts');
});
