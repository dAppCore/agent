<?php

// SPDX-License-Identifier: EUPL-1.2

declare(strict_types=1);

namespace Core\Mod\Agentic\Mcp\Services;

use Core\Mod\Agentic\Mcp\Data\ToolMetadata;
use Illuminate\Container\Container;
use InvalidArgumentException;

final class ToolRegistry
{
    /**
     * @var array<string, ToolMetadata>
     */
    private array $tools = [];

    public function __construct(
        private readonly ?Container $container = null,
    ) {}

    public static function registerSingleton(Container $container): self
    {
        if (! $container->bound(self::class)) {
            $container->singleton(self::class, fn (Container $app): self => new self($app));
        }

        return $container->make(self::class);
    }

    public function register(mixed $tool): ToolMetadata
    {
        $metadata = ToolMetadata::from($tool);

        if (isset($this->tools[$metadata->name])) {
            throw new InvalidArgumentException(sprintf(
                'Tool [%s] is already registered.',
                $metadata->name,
            ));
        }

        $this->tools[$metadata->name] = $metadata;

        return $metadata;
    }

    public function resolve(string $name): ?ToolMetadata
    {
        return $this->tools[$name] ?? null;
    }

    /**
     * @return array<int, ToolMetadata>
     */
    public function listTools(): array
    {
        return array_values($this->tools);
    }

    /**
     * @return array<string, array<int, string>>
     */
    public function buildDependencyGraph(): array
    {
        $graph = [];

        foreach ($this->tools as $name => $tool) {
            $graph[$name] = $tool->dependencyIdentifiers();
        }

        return $graph;
    }

    public function call(string $name, array $arguments = [], array $context = []): mixed
    {
        $tool = $this->resolve($name);
        if ($tool === null) {
            throw new InvalidArgumentException(sprintf(
                'Unknown tool [%s].',
                $name,
            ));
        }

        return $tool->call($arguments, $context);
    }
}
