<?php

// SPDX-License-Identifier: EUPL-1.2

declare(strict_types=1);

require_once dirname(__DIR__).'/Support/bootstrap.php';

mcpRequire('Mcp/Transport/McpContext.php');
mcpRequire('Mcp/Transport/Contracts/McpToolHandler.php');

use Core\Front\Mcp\Contracts\McpToolHandler;
use Core\Front\Mcp\McpContext;

test('McpToolHandler_schema_Good_can_be_implemented_with_the_expected_shape', function (): void {
    $handler = new class implements McpToolHandler
    {
        public static function schema(): array
        {
            return [
                'name' => 'list_posts',
                'description' => 'List CMS posts',
                'inputSchema' => ['type' => 'object'],
            ];
        }

        public function handle(array $args, McpContext $context): array
        {
            return ['ok' => true];
        }
    };

    expect($handler::schema())->toBe([
        'name' => 'list_posts',
        'description' => 'List CMS posts',
        'inputSchema' => ['type' => 'object'],
    ]);
});

test('McpToolHandler_handle_Bad_receives_the_transport_agnostic_context_object', function (): void {
    $context = new McpContext('sess-1');
    $handler = new class implements McpToolHandler
    {
        public static function schema(): array
        {
            return ['name' => 'ping', 'description' => 'Ping', 'inputSchema' => ['type' => 'object']];
        }

        public function handle(array $args, McpContext $context): array
        {
            return ['session_id' => $context->getSessionId()];
        }
    };

    expect($handler->handle([], $context)['session_id'])->toBe('sess-1');
});

test('McpToolHandler_interface_Ugly_exposes_exactly_the_two_contract_methods_required_by_the_rfc', function (): void {
    $methods = array_map(
        static fn (ReflectionMethod $method): string => $method->getName(),
        (new ReflectionClass(McpToolHandler::class))->getMethods(),
    );

    expect($methods)->toBe(['schema', 'handle']);
});
