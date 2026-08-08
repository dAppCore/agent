<?php

// SPDX-License-Identifier: EUPL-1.2

declare(strict_types=1);

use Core\Mod\Agentic\Mcp\Services\ToolDependencyService;
use Core\Mod\Agentic\Mcp\Tools\Agent\Contracts\AgentToolInterface;
use Core\Mod\Agentic\Services\AgentToolRegistry;
use Core\Mod\Agentic\Website\Mcp\Middleware\ValidateToolDependencies;
use Illuminate\Http\Request;

function mcpMiddlewareToolFixture(string $name, array $dependencies = []): object
{
    return new class($name, $dependencies) implements AgentToolInterface
    {
        public function __construct(
            private readonly string $toolName,
            private readonly array $toolDependencies,
        ) {}

        public function name(): string
        {
            return $this->toolName;
        }

        public function description(): string
        {
            return 'Middleware fixture tool';
        }

        public function inputSchema(): array
        {
            return ['type' => 'object'];
        }

        public function dependencies(): array
        {
            return $this->toolDependencies;
        }

        public function handle(array $arguments, array $context = []): array
        {
            return [
                'arguments' => $arguments,
                'context' => $context,
                'tool' => $this->toolName,
            ];
        }

        public function requiredScopes(): array
        {
            return ['read'];
        }

        public function category(): string
        {
            return 'testing';
        }
    };
}

test('ValidateToolDependencies_handle_Good_validates_json_rpc_tool_calls_and_records_successful_execution', function (): void {
    $registry = new AgentToolRegistry;
    $registry->register(mcpMiddlewareToolFixture('session_start'));
    $registry->register(mcpMiddlewareToolFixture('report_generate', [
        ['type' => 'tool', 'tool' => 'session_start', 'message' => 'Start session first.'],
        ['type' => 'context_exists', 'key' => 'workspace_id', 'message' => 'Workspace context required.'],
    ]));

    $service = new ToolDependencyService($registry, $this->app);
    $service->recordToolCall('sess-1', 'session_start');

    $middleware = new ValidateToolDependencies($service);
    $request = Request::create('/api/v1/mcp/tools/call', 'POST', [
        'method' => 'tools/call',
        'params' => [
            'name' => 'report_generate',
            'arguments' => [
                'session_id' => 'sess-1',
            ],
        ],
    ]);
    $request->attributes->set('workspace_id', 'workspace-1');
    $request->attributes->set('mcp_workspace_context', ['workspace_id' => 'workspace-1']);

    $response = $middleware->handle($request, fn () => response()->json(['success' => true]));

    expect($response->getStatusCode())->toBe(200)
        ->and($service->calledTools('sess-1'))->toBe(['session_start', 'report_generate']);
});

test('ValidateToolDependencies_handle_Bad_returns_conflict_when_required_dependencies_are_missing', function (): void {
    $registry = new AgentToolRegistry;
    $registry->register(mcpMiddlewareToolFixture('plan_list', [
        ['type' => 'context_exists', 'key' => 'workspace_id', 'message' => 'Workspace context required.'],
    ]));

    $service = new ToolDependencyService($registry, $this->app);
    $middleware = new ValidateToolDependencies($service);
    $request = Request::create('/api/v1/mcp/tools/call', 'POST', [
        'tool' => 'plan_list',
        'arguments' => [],
    ]);

    $response = $middleware->handle($request, fn () => response()->json(['success' => true]));
    $data = json_decode((string) $response->getContent(), true);

    expect($response->getStatusCode())->toBe(409)
        ->and($data['error'])->toBe('dependency_not_met')
        ->and($data['missing_dependencies'])->toHaveCount(1);
});

test('ValidateToolDependencies_handle_Ugly_converts_circular_dependency_failures_into_conflict_responses', function (): void {
    $registry = new AgentToolRegistry;
    $registry->register(mcpMiddlewareToolFixture('tool_alpha', [
        ['type' => 'tool', 'tool' => 'tool_bravo', 'message' => 'tool_bravo is required.'],
    ]));
    $registry->register(mcpMiddlewareToolFixture('tool_bravo', [
        ['type' => 'tool', 'tool' => 'tool_alpha', 'message' => 'tool_alpha is required.'],
    ]));

    $service = new ToolDependencyService($registry, $this->app);
    $middleware = new ValidateToolDependencies($service);
    $request = Request::create('/api/v1/mcp/tools/call', 'POST', [
        'tool' => 'tool_alpha',
        'arguments' => [
            'session_id' => 'sess-circular',
        ],
    ]);
    $request->attributes->set('workspace_id', 'workspace-1');

    $response = $middleware->handle($request, fn () => response()->json(['success' => true]));
    $data = json_decode((string) $response->getContent(), true);

    expect($response->getStatusCode())->toBe(409)
        ->and($data['error'])->toBe('dependency_validation_failed');
});
