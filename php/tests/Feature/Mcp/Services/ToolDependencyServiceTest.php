<?php

// SPDX-License-Identifier: EUPL-1.2

declare(strict_types=1);

use Core\Mod\Agentic\Mcp\Services\ToolDependencyService;
use Core\Mod\Agentic\Mcp\Tools\Agent\Contracts\AgentToolInterface;
use Core\Mod\Agentic\Services\AgentToolRegistry;

/**
 * A registrable tool declaring the given dependencies.
 *
 * An AgentToolInterface, not the loose array payload this used to build: the
 * registry these tests exercise is now the typed one, and a metadata array is
 * no longer a thing that can be registered. The dependency rows stay arrays,
 * which is what ToolDependencyService normalises.
 */
function mcpDependencyToolFixture(string $name, array $dependencies = []): AgentToolInterface
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
            return 'Fixture tool';
        }

        public function inputSchema(): array
        {
            return ['type' => 'object'];
        }

        public function dependencies(): array
        {
            return $this->toolDependencies;
        }

        public function handle(array $args, array $context = []): array
        {
            return [
                'arguments' => $args,
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

test('ToolDependencyService_validateDependencies_Good_walks_transitive_tool_graphs_before_execution', function (): void {
    $registry = new AgentToolRegistry;

    $registry->register(mcpDependencyToolFixture('session_start'));
    $registry->register(mcpDependencyToolFixture('session_log', [
        ['type' => 'tool_called', 'tool' => 'session_start', 'message' => 'Start session first.'],
        ['type' => 'session_state', 'key' => 'session_id', 'message' => 'Session context required.'],
    ]));
    $registry->register(mcpDependencyToolFixture('report_generate', [
        ['type' => 'tool', 'tool' => 'session_log', 'message' => 'Session logging must be available.'],
        ['type' => 'context_exists', 'key' => 'workspace_id', 'message' => 'Workspace context required.'],
    ]));

    $service = new ToolDependencyService($registry, $this->app);
    $service->recordToolCall('sess-1', 'session_start');
    $service->validateDependencies('sess-1', 'report_generate', [
        'workspace_id' => 'workspace-1',
        'session_id' => 'sess-1',
    ], []);

    expect($service->canExecute('report_generate', [
        'workspace_id' => 'workspace-1',
        'session_id' => 'sess-1',
    ], [], 'sess-1'))->toBeTrue();
});

test('ToolDependencyService_validateDependencies_Bad_reports_missing_context_requirements', function (): void {
    $registry = new AgentToolRegistry;
    $registry->register(mcpDependencyToolFixture('plan_list', [
        ['type' => 'context_exists', 'key' => 'workspace_id', 'message' => 'Workspace context required.'],
    ]));

    $service = new ToolDependencyService($registry, $this->app);
    $service->validateDependencies('plan_list', []);
})->throws(RuntimeException::class, 'Workspace context required.');

test('ToolDependencyService_validateDependencies_Ugly_detects_circular_tool_dependencies', function (): void {
    $registry = new AgentToolRegistry;

    $registry->register(mcpDependencyToolFixture('tool_alpha', [
        ['type' => 'tool', 'tool' => 'tool_bravo', 'message' => 'tool_bravo is required.'],
    ]));
    $registry->register(mcpDependencyToolFixture('tool_bravo', [
        ['type' => 'tool', 'tool' => 'tool_alpha', 'message' => 'tool_alpha is required.'],
    ]));

    $service = new ToolDependencyService($registry, $this->app);
    $service->validateDependencies('tool_alpha', []);
})->throws(RuntimeException::class, 'Circular dependency detected while validating [tool_alpha].');
