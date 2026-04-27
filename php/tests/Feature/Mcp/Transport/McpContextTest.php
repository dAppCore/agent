<?php

// SPDX-License-Identifier: EUPL-1.2

declare(strict_types=1);

require_once dirname(__DIR__).'/Support/bootstrap.php';

mcpRequire('Mcp/Transport/McpContext.php');

use Core\Front\Mcp\McpContext;
use Illuminate\Http\Request;

function mcpScopeSession(array $scopes = []): object
{
    return new class($scopes)
    {
        public function __construct(
            private readonly array $scopes,
        ) {}

        public function getApiKey(): object
        {
            return (object) ['permissions' => $this->scopes];
        }
    };
}

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

test('McpContext_getScopes_Good_returns_session_scopes', function (): void {
    $context = new McpContext(scopeSource: mcpScopeSession([
        'brain.remember',
        'brain.recall',
    ]));

    expect($context->getScopes())->toBe([
        'brain.remember',
        'brain.recall',
    ]);
});

test('McpContext_getScopes_Good_reads_authenticated_request_scopes_from_mcp_workspace_context', function (): void {
    $workspace = createWorkspace();
    $apiKey = createApiKey($workspace, 'Scoped MCP Key', [
        'brain.remember',
        'brain.recall',
    ]);

    $request = Request::create('/api/v1/mcp/tools/call', 'POST');
    $request->attributes->set('mcp_workspace_context', [
        'workspace_id' => $workspace->id,
        'api_key' => $apiKey,
    ]);

    $originalRequest = app()->bound('request') ? app('request') : null;
    app()->instance('request', $request);

    try {
        expect((new McpContext)->getScopes())->toBe([
            'brain.remember',
            'brain.recall',
        ]);
    } finally {
        if ($originalRequest instanceof Request) {
            app()->instance('request', $originalRequest);
        } else {
            app()->forgetInstance('request');
        }
    }
});

test('McpContext_hasScope_Good_returns_true_for_a_present_scope', function (): void {
    $context = new McpContext(scopeSource: mcpScopeSession([
        'brain.remember',
        'brain.recall',
    ]));

    expect($context->hasScope('brain.remember'))->toBeTrue();
});

test('McpContext_hasScope_Bad_returns_false_for_a_missing_scope', function (): void {
    $context = new McpContext(scopeSource: mcpScopeSession([
        'brain.remember',
        'brain.recall',
    ]));

    expect($context->hasScope('ofm.fan.read'))->toBeFalse();
});

test('McpContext_getScopes_Ugly_defaults_to_an_empty_array_for_an_empty_session', function (): void {
    $context = new McpContext(scopeSource: mcpScopeSession([]));

    expect($context->getScopes())->toBe([])
        ->and($context->hasScope('brain.remember'))->toBeFalse();
});
