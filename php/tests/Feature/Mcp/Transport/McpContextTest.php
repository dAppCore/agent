<?php

// SPDX-License-Identifier: EUPL-1.2

declare(strict_types=1);

require_once dirname(__DIR__).'/Support/bootstrap.php';

mcpRequire('Mcp/Transport/McpContext.php');

use Core\Front\Mcp\McpContext;

test('McpContext_getters_Good_track_the_current_session_and_plan', function (): void {
    $plan = (object) ['slug' => 'plan-1'];
    $context = new McpContext('sess-1', $plan);

    expect($context->getSessionId())->toBe('sess-1')
        ->and($context->getCurrentPlan())->toBe($plan)
        ->and($context->hasSession())->toBeTrue()
        ->and($context->hasPlan())->toBeTrue();
});

test('McpContext_callbacks_Bad_are_optional_and_can_be_left_unset', function (): void {
    $context = new McpContext;

    $context->sendNotification('mcp.progress', ['value' => 50]);
    $context->logToSession('noop');

    expect($context->hasSession())->toBeFalse()
        ->and($context->hasPlan())->toBeFalse();
});

test('McpContext_callbacks_Ugly_forward_notifications_and_session_logs_through_the_transport_hooks', function (): void {
    $captured = [];
    $context = new McpContext(
        notificationCallback: function (string $method, array $params) use (&$captured): void {
            $captured['notification'] = [$method, $params];
        },
        logCallback: function (string $message, string $type, array $data) use (&$captured): void {
            $captured['log'] = [$message, $type, $data];
        },
    );

    $context->sendNotification('mcp.progress', ['value' => 100]);
    $context->logToSession('finished', 'info', ['ok' => true]);

    expect($captured['notification'])->toBe(['mcp.progress', ['value' => 100]])
        ->and($captured['log'])->toBe(['finished', 'info', ['ok' => true]]);
});
