<?php

declare(strict_types=1);

namespace Core\Mod\Agentic\Mcp\Tools\Agent;

use Closure;
use Core\Mcp\Dependencies\HasDependencies;
use Core\Mcp\Exceptions\CircuitOpenException;
use Core\Mcp\Services\CircuitBreaker;
use Core\Mcp\Tools\Concerns\ValidatesDependencies;
use Core\Mod\Agentic\Mcp\Tools\Agent\Contracts\AgentToolInterface;

/**
 * Base class for MCP Agent Server tools.
 *
 * Provides common functionality for all extracted agent tools.
 */
abstract class AgentTool implements AgentToolInterface, HasDependencies
{
    use ValidatesDependencies;

    /**
     * Tool category for grouping in the registry.
     */
    protected string $category = 'general';

    /**
     * Required permission scopes.
     *
     * @var array<string>
     */
    protected array $scopes = ['read'];

    /**
     * Tool-specific timeout override (null uses config default).
     */
    protected ?int $timeout = null;

    /**
     * Get the tool category.
     */
    public function category(): string
    {
        return $this->category;
    }

    /**
     * Get required scopes.
     */
    public function requiredScopes(): array
    {
        return $this->scopes;
    }

    /**
     * Get the timeout for this tool in seconds.
     */
    public function getTimeout(): int
    {
        // Check tool-specific override
        if ($this->timeout !== null) {
            return $this->timeout;
        }

        // Check per-tool config
        $perToolTimeout = config('mcp.timeouts.per_tool.'.$this->name());
        if ($perToolTimeout !== null) {
            return (int) $perToolTimeout;
        }

        // Use default timeout
        return (int) config('mcp.timeouts.default', 30);
    }

    /**
     * Convert to MCP tool definition format.
     */
    public function toMcpDefinition(): array
    {
        return [
            'name' => $this->name(),
            'description' => $this->description(),
            'inputSchema' => $this->inputSchema(),
        ];
    }

    /**
     * Create a success response.
     */
    protected function success(array $data): array
    {
        return array_merge(['success' => true], $data);
    }

    /**
     * Create an error response.
     */
    protected function error(string $message, ?string $code = null): array
    {
        $response = ['error' => $message];

        if ($code !== null) {
            $response['code'] = $code;
        }

        return $response;
    }

    /**
     * Get a required argument or return error.
     */
    protected function require(array $args, string $key, ?string $label = null): mixed
    {
        if (! isset($args[$key]) || $args[$key] === '') {
            throw new \InvalidArgumentException(
                sprintf('%s is required', $label ?? $key)
            );
        }

        return $args[$key];
    }

    /**
     * Get an optional argument with default.
     */
    protected function optional(array $args, string $key, mixed $default = null): mixed
    {
        return $args[$key] ?? $default;
    }

    /**
     * Validate and get a required string argument.
     *
     * @throws \InvalidArgumentException
     */
    protected function requireString(array $args, string $key, ?int $maxLength = null, ?string $label = null): string
    {
        $value = $this->require($args, $key, $label);

        if (! is_string($value)) {
            throw new \InvalidArgumentException(
                sprintf('%s must be a string', $label ?? $key)
            );
        }

        if ($maxLength !== null && strlen($value) > $maxLength) {
            throw new \InvalidArgumentException(
                sprintf('%s exceeds maximum length of %d characters', $label ?? $key, $maxLength)
            );
        }

        return $value;
    }

    /**
     * Validate and get a required integer argument.
     *
     * @throws \InvalidArgumentException
     */
    protected function requireInt(array $args, string $key, ?int $min = null, ?int $max = null, ?string $label = null): int
    {
        $value = $this->require($args, $key, $label);

        if (! is_int($value) && ! (is_numeric($value) && (int) $value == $value)) {
            throw new \InvalidArgumentException(
                sprintf('%s must be an integer', $label ?? $key)
            );
        }

        $intValue = (int) $value;

        if ($min !== null && $intValue < $min) {
            throw new \InvalidArgumentException(
                sprintf('%s must be at least %d', $label ?? $key, $min)
            );
        }

        if ($max !== null && $intValue > $max) {
            throw new \InvalidArgumentException(
                sprintf('%s must be at most %d', $label ?? $key, $max)
            );
        }

        return $intValue;
    }

    /**
     * Validate and get an optional string argument.
     */
    protected function optionalString(array $args, string $key, ?string $default = null, ?int $maxLength = null): ?string
    {
        $value = $args[$key] ?? $default;

        if ($value === null) {
            return null;
        }

        if (! is_string($value)) {
            throw new \InvalidArgumentException(
                sprintf('%s must be a string', $key)
            );
        }

        if ($maxLength !== null && strlen($value) > $maxLength) {
            throw new \InvalidArgumentException(
                sprintf('%s exceeds maximum length of %d characters', $key, $maxLength)
            );
        }

        return $value;
    }

    /**
     * Validate and get an optional integer argument.
     */
    protected function optionalInt(array $args, string $key, ?int $default = null, ?int $min = null, ?int $max = null): ?int
    {
        if (! isset($args[$key])) {
            return $default;
        }

        $value = $args[$key];

        if (! is_int($value) && ! (is_numeric($value) && (int) $value == $value)) {
            throw new \InvalidArgumentException(
                sprintf('%s must be an integer', $key)
            );
        }

        $intValue = (int) $value;

        if ($min !== null && $intValue < $min) {
            throw new \InvalidArgumentException(
                sprintf('%s must be at least %d', $key, $min)
            );
        }

        if ($max !== null && $intValue > $max) {
            throw new \InvalidArgumentException(
                sprintf('%s must be at most %d', $key, $max)
            );
        }

        return $intValue;
    }

    /**
     * Validate and get a required array argument.
     *
     * @throws \InvalidArgumentException
     */
    protected function requireArray(array $args, string $key, ?string $label = null): array
    {
        $value = $this->require($args, $key, $label);

        if (! is_array($value)) {
            throw new \InvalidArgumentException(
                sprintf('%s must be an array', $label ?? $key)
            );
        }

        return $value;
    }

    /**
     * Validate a value is one of the allowed values.
     *
     * @throws \InvalidArgumentException
     */
    protected function requireEnum(array $args, string $key, array $allowed, ?string $label = null): string
    {
        $value = $this->requireString($args, $key, null, $label);

        if (! in_array($value, $allowed, true)) {
            throw new \InvalidArgumentException(
                sprintf('%s must be one of: %s', $label ?? $key, implode(', ', $allowed))
            );
        }

        return $value;
    }

    /**
     * Validate an optional enum value.
     */
    protected function optionalEnum(array $args, string $key, array $allowed, ?string $default = null): ?string
    {
        if (! isset($args[$key])) {
            return $default;
        }

        $value = $args[$key];

        if (! is_string($value)) {
            throw new \InvalidArgumentException(
                sprintf('%s must be a string', $key)
            );
        }

        if (! in_array($value, $allowed, true)) {
            throw new \InvalidArgumentException(
                sprintf('%s must be one of: %s', $key, implode(', ', $allowed))
            );
        }

        return $value;
    }

    /**
     * Execute an operation with circuit breaker protection.
     *
     * Wraps calls to external modules (Agentic, Content, etc.) with fault tolerance.
     * If the service fails repeatedly, the circuit opens and returns the fallback.
     *
     * @param  string  $service  Service identifier (e.g., 'agentic', 'content')
     * @param  Closure  $operation  The operation to execute
     * @param  Closure|null  $fallback  Optional fallback when circuit is open
     * @return mixed The operation result or fallback value
     */
    protected function withCircuitBreaker(string $service, Closure $operation, ?Closure $fallback = null): mixed
    {
        $breaker = app(CircuitBreaker::class);

        try {
            return $breaker->call($service, $operation, $fallback);
        } catch (CircuitOpenException $e) {
            // If no fallback was provided and circuit is open, return error response
            return $this->error($e->getMessage(), 'service_unavailable');
        }
    }

    /**
     * Check if an external service is available.
     *
     * @param  string  $service  Service identifier (e.g., 'agentic', 'content')
     */
    protected function isServiceAvailable(string $service): bool
    {
        return app(CircuitBreaker::class)->isAvailable($service);
    }
}
