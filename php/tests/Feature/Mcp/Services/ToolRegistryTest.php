<?php

// SPDX-License-Identifier: EUPL-1.2

declare(strict_types=1);

use Core\Mod\Agentic\Mcp\Services\ToolRegistry;

function mcpToolRegistryFixture(string $name, array $dependencies = []): object
{
    return new class($name, $dependencies)
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

        public function handle(array $arguments, array $context = []): array
        {
            return [
                'arguments' => $arguments,
                'context' => $context,
                'tool' => $this->toolName,
            ];
        }
    };
}

test('ToolRegistry_register_resolve_listTools_buildDependencyGraph_Good_binds_the_registry_as_a_singleton', function (): void {
    $registry = ToolRegistry::registerSingleton($this->app);

    $registry->register(mcpToolRegistryFixture('session_start'));
    $registry->register(mcpToolRegistryFixture('report_generate', [
        ['type' => 'tool', 'tool' => 'session_start'],
        ['type' => 'context_exists', 'key' => 'workspace_id'],
    ]));

    $resolved = $this->app->make(ToolRegistry::class);
    $graph = $registry->buildDependencyGraph();

    expect($resolved)->toBe($registry)
        ->and($registry->resolve('report_generate')?->name)->toBe('report_generate')
        ->and(array_map(
            static fn ($tool): string => $tool->name,
            $registry->listTools(),
        ))->toBe(['session_start', 'report_generate'])
        ->and($graph)->toBe([
            'session_start' => [],
            'report_generate' => ['session_start', 'workspace_id'],
        ])
        ->and($registry->call('report_generate', ['draft' => true], ['workspace_id' => 'ws-1']))
        ->toBe([
            'arguments' => ['draft' => true],
            'context' => ['workspace_id' => 'ws-1'],
            'tool' => 'report_generate',
        ]);
});

test('ToolRegistry_register_Bad_rejects_duplicate_tool_names', function (): void {
    $registry = new ToolRegistry;

    $registry->register(mcpToolRegistryFixture('session_start'));
    $registry->register(mcpToolRegistryFixture('session_start'));
})->throws(InvalidArgumentException::class, 'Tool [session_start] is already registered.');

test('ToolRegistry_register_Ugly_rejects_payloads_without_a_callable_handler', function (): void {
    $registry = new ToolRegistry;

    $registry->register([
        'name' => 'broken_tool',
        'description' => 'No callable handler',
    ]);
})->throws(InvalidArgumentException::class, 'A callable handler is required');
