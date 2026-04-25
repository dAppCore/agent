<?php

// SPDX-License-Identifier: EUPL-1.2

declare(strict_types=1);

use Core\Mod\Agentic\Mcp\Services\ToolDependencyService;
use Core\Mod\Agentic\Mcp\Services\ToolRegistry;

function mcpDependencyToolFixture(string $name, array $dependencies = []): array
{
    return [
        'name' => $name,
        'description' => 'Fixture tool',
        'dependencies' => $dependencies,
        'handler' => static fn (array $arguments = [], array $context = []): array => [
            'arguments' => $arguments,
            'context' => $context,
            'tool' => $name,
        ],
    ];
}

test('ToolDependencyService_validateDependencies_Good_walks_transitive_tool_graphs_before_execution', function (): void {
    $registry = new ToolRegistry;

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
    $registry = new ToolRegistry;
    $registry->register(mcpDependencyToolFixture('plan_list', [
        ['type' => 'context_exists', 'key' => 'workspace_id', 'message' => 'Workspace context required.'],
    ]));

    $service = new ToolDependencyService($registry, $this->app);
    $service->validateDependencies('plan_list', []);
})->throws(RuntimeException::class, 'Workspace context required.');

test('ToolDependencyService_validateDependencies_Ugly_detects_circular_tool_dependencies', function (): void {
    $registry = new ToolRegistry;

    $registry->register(mcpDependencyToolFixture('tool_alpha', [
        ['type' => 'tool', 'tool' => 'tool_bravo', 'message' => 'tool_bravo is required.'],
    ]));
    $registry->register(mcpDependencyToolFixture('tool_bravo', [
        ['type' => 'tool', 'tool' => 'tool_alpha', 'message' => 'tool_alpha is required.'],
    ]));

    $service = new ToolDependencyService($registry, $this->app);
    $service->validateDependencies('tool_alpha', []);
})->throws(RuntimeException::class, 'Circular dependency detected while validating [tool_alpha].');
