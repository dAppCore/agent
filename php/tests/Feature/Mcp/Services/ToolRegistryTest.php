<?php

// SPDX-License-Identifier: EUPL-1.2

declare(strict_types=1);

use Core\Mod\Agentic\Mcp\Tools\Agent\Contracts\AgentToolInterface;
use Core\Mod\Agentic\Services\AgentToolRegistry;

function mcpToolRegistryFixture(string $name, array $dependencies = []): object
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

test('AgentToolRegistry_register_resolve_listTools_buildDependencyGraph_Good_absorbs_the_mcp_surface', function (): void {
    // A fresh registry, not the container's: Boot now fills the singleton with
    // the forty real tools, so registering session_start into it would collide
    // with the real one and listTools() would return forty-two.
    $registry = new AgentToolRegistry;

    $registry->register(mcpToolRegistryFixture('session_start'));
    $registry->register(mcpToolRegistryFixture('report_generate', [
        ['type' => 'tool', 'tool' => 'session_start'],
        ['type' => 'context_exists', 'key' => 'workspace_id'],
    ]));

    $graph = $registry->buildDependencyGraph();

    expect($this->app->make(AgentToolRegistry::class))
        ->toBe($this->app->make(AgentToolRegistry::class))
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

test('AgentToolRegistry_register_Bad_rejects_duplicate_tool_names', function (): void {
    $registry = new AgentToolRegistry;

    $registry->register(mcpToolRegistryFixture('session_start'));
    $registry->register(mcpToolRegistryFixture('session_start'));
})->throws(InvalidArgumentException::class, 'Tool [session_start] is already registered.');
