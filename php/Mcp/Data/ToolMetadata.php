<?php

// SPDX-License-Identifier: EUPL-1.2

declare(strict_types=1);

namespace Core\Mod\Agentic\Mcp\Data;

use Closure;
use InvalidArgumentException;

final readonly class ToolMetadata
{
    public function __construct(
        public string $name,
        public Closure $handler,
        public string $description = '',
        public array $inputSchema = [],
        public array $dependencies = [],
        public array $metadata = [],
    ) {}

    public static function from(mixed $tool): self
    {
        if ($tool instanceof self) {
            return $tool;
        }

        if (is_array($tool)) {
            $name = (string) ($tool['name'] ?? $tool['id'] ?? '');
            if ($name === '') {
                throw new InvalidArgumentException('Tool registrations require a name.');
            }

            return new self(
                name: $name,
                handler: self::normaliseHandler($tool['handler'] ?? $tool['callable'] ?? null),
                description: (string) ($tool['description'] ?? ''),
                inputSchema: (array) ($tool['inputSchema'] ?? $tool['schema'] ?? []),
                dependencies: array_values((array) ($tool['dependencies'] ?? [])),
                metadata: array_diff_key($tool, array_flip([
                    'id',
                    'name',
                    'handler',
                    'callable',
                    'description',
                    'inputSchema',
                    'schema',
                    'dependencies',
                ])),
            );
        }

        if (! is_object($tool)) {
            throw new InvalidArgumentException('Tool registrations must be arrays, objects, or ToolMetadata instances.');
        }

        $name = self::extractString($tool, 'name');

        return new self(
            name: $name,
            handler: self::normaliseHandler(self::extractHandler($tool)),
            description: self::extractOptionalString($tool, 'description'),
            inputSchema: self::extractOptionalArray($tool, 'inputSchema'),
            dependencies: self::extractOptionalList($tool, 'dependencies'),
            metadata: ['class' => $tool::class],
        );
    }

    public function call(array $arguments = [], array $context = []): mixed
    {
        try {
            return ($this->handler)($arguments, $context);
        } catch (\ArgumentCountError) {
            return ($this->handler)($arguments);
        }
    }

    /**
     * @return array<string>
     */
    public function dependencyIdentifiers(): array
    {
        $identifiers = array_map(
            fn (mixed $dependency): string => self::dependencyIdentifier($dependency),
            $this->dependencies
        );

        return array_values(array_unique(array_filter($identifiers)));
    }

    private static function extractHandler(object $tool): mixed
    {
        $vars = get_object_vars($tool);

        if (method_exists($tool, 'handle')) {
            return [$tool, 'handle'];
        }

        if (array_key_exists('handler', $vars)) {
            return $vars['handler'];
        }

        if (is_callable($tool)) {
            return $tool;
        }

        throw new InvalidArgumentException(sprintf(
            'Tool [%s] must expose a handle() method or a callable handler.',
            $tool::class,
        ));
    }

    private static function extractString(object $tool, string $member): string
    {
        $value = self::extractValue($tool, $member);
        if (! is_string($value) || $value === '') {
            throw new InvalidArgumentException(sprintf(
                'Tool [%s] must return a non-empty string from %s().',
                $tool::class,
                $member,
            ));
        }

        return $value;
    }

    private static function extractOptionalString(object $tool, string $member): string
    {
        $value = self::extractValue($tool, $member, '');

        return is_string($value) ? $value : '';
    }

    /**
     * @return array<mixed>
     */
    private static function extractOptionalArray(object $tool, string $member): array
    {
        $value = self::extractValue($tool, $member, []);

        return is_array($value) ? $value : [];
    }

    /**
     * @return array<int, mixed>
     */
    private static function extractOptionalList(object $tool, string $member): array
    {
        $value = self::extractValue($tool, $member, []);

        return is_array($value) ? array_values($value) : [];
    }

    private static function extractValue(object $tool, string $member, mixed $default = null): mixed
    {
        if (method_exists($tool, $member)) {
            return $tool->{$member}();
        }

        $vars = get_object_vars($tool);

        return $vars[$member] ?? $default;
    }

    private static function normaliseHandler(mixed $handler): Closure
    {
        if (! is_callable($handler)) {
            throw new InvalidArgumentException('A callable handler is required for each tool registration.');
        }

        return $handler instanceof Closure
            ? $handler
            : Closure::fromCallable($handler);
    }

    private static function dependencyIdentifier(mixed $dependency): string
    {
        if (is_string($dependency)) {
            return $dependency;
        }

        if (is_array($dependency)) {
            return (string) ($dependency['tool']
                ?? $dependency['key']
                ?? $dependency['abstract']
                ?? $dependency['name']
                ?? '');
        }

        if (! is_object($dependency)) {
            return '';
        }

        $vars = get_object_vars($dependency);

        if (array_key_exists('tool', $vars)) {
            return (string) $vars['tool'];
        }

        if (array_key_exists('key', $vars)) {
            return (string) $vars['key'];
        }

        if (array_key_exists('abstract', $vars)) {
            return (string) $vars['abstract'];
        }

        if (array_key_exists('name', $vars)) {
            return (string) $vars['name'];
        }

        return '';
    }
}
