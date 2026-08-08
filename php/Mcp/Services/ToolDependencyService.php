<?php

// SPDX-License-Identifier: EUPL-1.2

declare(strict_types=1);

namespace Core\Mod\Agentic\Mcp\Services;

use Carbon\CarbonImmutable;
use Core\Mcp\Dependencies\ToolDependency;
use Core\Mod\Agentic\Services\AgentToolRegistry;
use Illuminate\Container\Container;
use InvalidArgumentException;
use RuntimeException;

final class ToolDependencyService
{
    /**
     * @var array<string, array<int, mixed>>
     */
    private array $dependencies = [];

    /**
     * @var array<string, array<string, array{tool: string, payload: array, recorded_at: string}>>
     */
    private array $toolCalls = [];

    public function __construct(
        private ?AgentToolRegistry $registry = null,
        private readonly ?Container $container = null,
    ) {
        $this->registry ??= $this->resolveRegistry();
    }

    public function register(string $toolId, array $dependencies): void
    {
        $this->dependencies[$toolId] = array_values($dependencies);
    }

    /**
     * @return array<int, mixed>
     */
    public function dependenciesFor(string $toolId): array
    {
        if (array_key_exists($toolId, $this->dependencies)) {
            return $this->dependencies[$toolId];
        }

        return $this->registry?->resolve($toolId)?->dependencies ?? [];
    }

    public function canExecute(
        string $toolId,
        array $context = [],
        array $arguments = [],
        ?string $sessionId = null,
    ): bool {
        return $this->missing($toolId, $context, $arguments, $sessionId) === [];
    }

    /**
     * @return array<int, array{tool: string, type: string, key: string, message: string}>
     */
    public function missing(
        string $toolId,
        array $context = [],
        array $arguments = [],
        ?string $sessionId = null,
    ): array {
        $resolvedSession = $sessionId ?? (string) ($context['session_id'] ?? 'anonymous');

        return $this->missingForTool($toolId, $context, $arguments, $resolvedSession, []);
    }

    public function validateDependencies(mixed ...$arguments): void
    {
        [$sessionId, $toolId, $context, $payload] = $this->parseValidationArguments($arguments);
        $missing = $this->missing($toolId, $context, $payload, $sessionId);

        if ($missing === []) {
            return;
        }

        $messages = array_map(
            static fn (array $dependency): string => $dependency['message'],
            $missing,
        );

        throw $this->buildMissingDependencyException($toolId, $messages);
    }

    public function recordToolCall(string $sessionId, string $toolId, array $payload = []): void
    {
        $this->toolCalls[$sessionId][$toolId] = [
            'tool' => $toolId,
            'payload' => $payload,
            'recorded_at' => CarbonImmutable::now()->toIso8601String(),
        ];
    }

    /**
     * @return array<int, string>
     */
    public function calledTools(string $sessionId): array
    {
        return array_keys($this->toolCalls[$sessionId] ?? []);
    }

    private function resolveRegistry(): ?AgentToolRegistry
    {
        $container = $this->container ?? Container::getInstance();

        if (! $container instanceof Container || ! $container->bound(AgentToolRegistry::class)) {
            return null;
        }

        return $container->make(AgentToolRegistry::class);
    }

    /**
     * @param  array<int, mixed>  $arguments
     * @return array{0: string, 1: string, 2: array, 3: array}
     */
    private function parseValidationArguments(array $arguments): array
    {
        if (count($arguments) === 0) {
            throw new InvalidArgumentException('validateDependencies() requires at least a tool identifier.');
        }

        if (count($arguments) >= 2 && is_string($arguments[1])) {
            return [
                (string) $arguments[0],
                (string) $arguments[1],
                is_array($arguments[2] ?? null) ? $arguments[2] : [],
                is_array($arguments[3] ?? null) ? $arguments[3] : [],
            ];
        }

        $context = is_array($arguments[1] ?? null) ? $arguments[1] : [];
        $sessionId = is_string($context['session_id'] ?? null)
            ? $context['session_id']
            : 'anonymous';

        return [
            $sessionId,
            (string) $arguments[0],
            $context,
            is_array($arguments[2] ?? null) ? $arguments[2] : [],
        ];
    }

    /**
     * @param  array<string, bool>  $trail
     * @return array<int, array{tool: string, type: string, key: string, message: string}>
     */
    private function missingForTool(
        string $toolId,
        array $context,
        array $arguments,
        string $sessionId,
        array $trail,
    ): array {
        if (isset($trail[$toolId])) {
            throw new RuntimeException(sprintf(
                'Circular dependency detected while validating [%s].',
                $toolId,
            ));
        }

        $trail[$toolId] = true;
        $missing = [];

        foreach ($this->dependenciesFor($toolId) as $dependency) {
            $normalised = $this->normaliseDependency($dependency);

            if (($normalised['optional'] ?? false) === true) {
                continue;
            }

            $resolved = $this->resolveDependency($toolId, $normalised, $context, $arguments, $sessionId, $trail);
            if ($resolved !== null) {
                $missing[] = $resolved;
            }
        }

        return $missing;
    }

    /**
     * @param  array<string, mixed>  $dependency
     * @param  array<string, bool>  $trail
     * @return array{tool: string, type: string, key: string, message: string}|null
     */
    private function resolveDependency(
        string $toolId,
        array $dependency,
        array $context,
        array $arguments,
        string $sessionId,
        array $trail,
    ): ?array {
        $family = $this->dependencyFamily($dependency['type']);
        $key = $dependency['key'];
        $message = $dependency['message'];
        $options = $dependency['options'];

        return match ($family) {
            'context' => $this->contextDependencyMissing($toolId, $key, $message, $context, $options),
            'session' => $this->sessionDependencyMissing($toolId, $key, $message, $context),
            'entity' => $this->entityDependencyMissing($toolId, $key, $message, $context, $arguments, $options),
            'tool' => $this->toolDependencyMissing(
                $toolId,
                $dependency['type'],
                $key,
                $message,
                $context,
                $arguments,
                $sessionId,
                $trail,
            ),
            'config' => $this->configDependencyMissing($toolId, $key, $message),
            'container' => $this->containerDependencyMissing($toolId, $key, $message),
            'callback' => $this->callbackDependencyMissing($toolId, $key, $message, $context, $arguments, $options),
            default => [
                'tool' => $toolId,
                'type' => $dependency['type'],
                'key' => $key,
                'message' => $message,
            ],
        };
    }

    /**
     * @return array<string, mixed>
     */
    private function normaliseDependency(mixed $dependency): array
    {
        if (is_string($dependency)) {
            return [
                'type' => $this->guessStringType($dependency),
                'key' => $dependency,
                'message' => sprintf('Dependency [%s] is required.', $dependency),
                'optional' => false,
                'options' => [],
            ];
        }

        if (is_array($dependency)) {
            $type = (string) ($dependency['type']
                ?? (array_key_exists('tool', $dependency) ? 'tool' : null)
                ?? (array_key_exists('abstract', $dependency) ? 'container' : null)
                ?? (array_key_exists('callback', $dependency) ? 'callback' : null)
                ?? (array_key_exists('config', $dependency) ? 'config' : null)
                ?? (array_key_exists('arg_key', $dependency) ? 'entity_exists' : 'context_exists'));
            $key = (string) ($dependency['key']
                ?? $dependency['tool']
                ?? $dependency['abstract']
                ?? $dependency['config']
                ?? $dependency['name']
                ?? '');

            return [
                'type' => $type,
                'key' => $key,
                'message' => (string) ($dependency['message'] ?? sprintf('Dependency [%s] is required.', $key)),
                'optional' => (bool) ($dependency['optional'] ?? false),
                'options' => array_merge(
                    (array) ($dependency['options'] ?? []),
                    array_filter([
                        'arg_key' => $dependency['arg_key'] ?? null,
                        'callback' => $dependency['callback'] ?? null,
                    ], static fn (mixed $value): bool => $value !== null),
                ),
            ];
        }

        if ($dependency instanceof ToolDependency) {
            // toArray(), not get_object_vars(): the object holds $type as a
            // DependencyType enum and names its text $description. The array
            // branch below casts type to string — which fatals on an enum — and
            // looks for 'message', so a raw property dump both breaks and
            // silently loses the description.
            $fields = $dependency->toArray();
            $fields['message'] = $fields['description'] ?? null;

            return $this->normaliseDependency(array_filter(
                $fields,
                static fn (mixed $value): bool => $value !== null,
            ));
        }

        if (is_object($dependency)) {
            $vars = get_object_vars($dependency);

            return $this->normaliseDependency($vars);
        }

        throw new InvalidArgumentException('Unsupported dependency definition supplied.');
    }

    private function guessStringType(string $dependency): string
    {
        if ($this->registry?->resolve($dependency) !== null || array_key_exists($dependency, $this->dependencies)) {
            return 'tool';
        }

        $container = $this->container ?? Container::getInstance();
        if ($container instanceof Container && ($container->bound($dependency) || class_exists($dependency))) {
            return 'container';
        }

        return 'context_exists';
    }

    private function dependencyFamily(string $type): string
    {
        $normalised = strtolower($type);

        return match (true) {
            str_contains($normalised, 'context') => 'context',
            str_contains($normalised, 'session') => 'session',
            str_contains($normalised, 'entity') => 'entity',
            str_contains($normalised, 'tool') => 'tool',
            str_contains($normalised, 'config') => 'config',
            str_contains($normalised, 'container'),
            str_contains($normalised, 'binding'),
            str_contains($normalised, 'service') => 'container',
            str_contains($normalised, 'callback') => 'callback',
            default => 'unknown',
        };
    }

    /**
     * @return array{tool: string, type: string, key: string, message: string}|null
     */
    private function contextDependencyMissing(
        string $toolId,
        string $key,
        string $message,
        array $context,
        array $options,
    ): ?array {
        $value = data_get($context, $key);

        if ($value === null || $value === '') {
            return [
                'tool' => $toolId,
                'type' => 'context_exists',
                'key' => $key,
                'message' => $message,
            ];
        }

        if (array_key_exists('value', $options) && $options['value'] !== $value) {
            return [
                'tool' => $toolId,
                'type' => 'context_exists',
                'key' => $key,
                'message' => $message,
            ];
        }

        return null;
    }

    /**
     * @return array{tool: string, type: string, key: string, message: string}|null
     */
    private function sessionDependencyMissing(
        string $toolId,
        string $key,
        string $message,
        array $context,
    ): ?array {
        $value = data_get($context, $key);

        if ($value === null || $value === '') {
            return [
                'tool' => $toolId,
                'type' => 'session_state',
                'key' => $key,
                'message' => $message,
            ];
        }

        $activeSessions = (array) ($context['active_sessions'] ?? []);
        if ($activeSessions !== [] && ! in_array((string) $value, $activeSessions, true)) {
            return [
                'tool' => $toolId,
                'type' => 'session_state',
                'key' => $key,
                'message' => $message,
            ];
        }

        return null;
    }

    /**
     * @return array{tool: string, type: string, key: string, message: string}|null
     */
    private function entityDependencyMissing(
        string $toolId,
        string $key,
        string $message,
        array $context,
        array $arguments,
        array $options,
    ): ?array {
        $argKey = (string) ($options['arg_key'] ?? $key.'_id');
        $identifier = $arguments[$argKey] ?? $context[$argKey] ?? null;

        if ($identifier === null || $identifier === '') {
            return [
                'tool' => $toolId,
                'type' => 'entity_exists',
                'key' => $key,
                'message' => $message,
            ];
        }

        $resolver = $context['entity_exists'] ?? null;
        if (is_callable($resolver)) {
            $exists = (bool) $resolver($key, $identifier, $options, $context, $arguments);

            return $exists ? null : [
                'tool' => $toolId,
                'type' => 'entity_exists',
                'key' => $key,
                'message' => $message,
            ];
        }

        $entities = (array) ($context['entities'][$key] ?? []);
        $exists = array_key_exists((string) $identifier, $entities)
            || in_array($identifier, $entities, true);

        return $exists ? null : [
            'tool' => $toolId,
            'type' => 'entity_exists',
            'key' => $key,
            'message' => $message,
        ];
    }

    /**
     * @return array{tool: string, type: string, key: string, message: string}|null
     */
    private function toolDependencyMissing(
        string $toolId,
        string $type,
        string $key,
        string $message,
        array $context,
        array $arguments,
        string $sessionId,
        array $trail,
    ): ?array {
        $requiresPreviousCall = str_contains(strtolower($type), 'called');

        if (($this->registry?->resolve($key) === null) && ! array_key_exists($key, $this->dependencies)) {
            return [
                'tool' => $toolId,
                'type' => $type,
                'key' => $key,
                'message' => $message,
            ];
        }

        if ($requiresPreviousCall && ! $this->wasToolCalled($sessionId, $key, $context)) {
            return [
                'tool' => $toolId,
                'type' => $type,
                'key' => $key,
                'message' => $message,
            ];
        }

        $nestedMissing = $this->missingForTool($key, $context, $arguments, $sessionId, $trail);

        return $nestedMissing === [] ? null : $nestedMissing[0];
    }

    /**
     * @return array{tool: string, type: string, key: string, message: string}|null
     */
    private function configDependencyMissing(string $toolId, string $key, string $message): ?array
    {
        $value = config($key);

        return ($value === null || $value === '')
            ? [
                'tool' => $toolId,
                'type' => 'config',
                'key' => $key,
                'message' => $message,
            ]
            : null;
    }

    /**
     * @return array{tool: string, type: string, key: string, message: string}|null
     */
    private function containerDependencyMissing(string $toolId, string $key, string $message): ?array
    {
        $container = $this->container ?? Container::getInstance();
        $isBound = $container instanceof Container
            && ($container->bound($key) || class_exists($key));

        return $isBound ? null : [
            'tool' => $toolId,
            'type' => 'container',
            'key' => $key,
            'message' => $message,
        ];
    }

    /**
     * @return array{tool: string, type: string, key: string, message: string}|null
     */
    private function callbackDependencyMissing(
        string $toolId,
        string $key,
        string $message,
        array $context,
        array $arguments,
        array $options,
    ): ?array {
        $callback = $options['callback'] ?? null;

        if (! is_callable($callback)) {
            return [
                'tool' => $toolId,
                'type' => 'callback',
                'key' => $key,
                'message' => $message,
            ];
        }

        return $callback($context, $arguments, $key) === true ? null : [
            'tool' => $toolId,
            'type' => 'callback',
            'key' => $key,
            'message' => $message,
        ];
    }

    private function wasToolCalled(string $sessionId, string $toolId, array $context): bool
    {
        if (isset($this->toolCalls[$sessionId][$toolId])) {
            return true;
        }

        return in_array($toolId, (array) ($context['called_tools'] ?? []), true);
    }

    /**
     * @param  array<int, string>  $messages
     */
    private function buildMissingDependencyException(string $toolId, array $messages): \Throwable
    {
        $message = sprintf(
            'Dependencies not met for [%s]: %s',
            $toolId,
            implode('; ', $messages),
        );

        $exceptionClass = 'Core\\Mcp\\Exceptions\\MissingDependencyException';
        if (class_exists($exceptionClass)) {
            return new $exceptionClass($message);
        }

        return new RuntimeException($message);
    }
}
