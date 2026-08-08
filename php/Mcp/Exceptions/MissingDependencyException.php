<?php

// SPDX-License-Identifier: EUPL-1.2

declare(strict_types=1);

namespace Core\Mcp\Exceptions;

use RuntimeException;

/**
 * Thrown when a tool's declared dependencies are not satisfied.
 *
 * Shaped for this repo's ToolDependencyService, not copied from dappcore/mcp's
 * version of the same class. That one takes
 * (string $toolName, array $missingDependencies, array $suggestedOrder) and
 * builds its own message, while this service raises it with a single,
 * already-composed message:
 *
 *     $exceptionClass = 'Core\Mcp\Exceptions\MissingDependencyException';
 *     if (class_exists($exceptionClass)) {
 *         return new $exceptionClass($message);
 *     }
 *
 * Rolling the upstream signature in verbatim would have turned a class-not-found
 * into an ArgumentCountError the first time a dependency went unmet — a
 * different fatal, not a fix. The extra detail stays available as optional
 * constructor arguments, so callers that have it can pass it.
 */
class MissingDependencyException extends RuntimeException
{
    /**
     * @param  array<int, array{tool: string, type: string, key: string, message: string}>  $missingDependencies
     *                                                                                                            The rows ToolDependencyService::missing() returns — arrays, not
     *                                                                                                            ToolDependency objects, because that is what this service produces.
     * @param  array<int, string>  $suggestedOrder  Tools worth calling first.
     */
    public function __construct(
        string $message,
        public readonly array $missingDependencies = [],
        public readonly string $toolName = '',
        public readonly array $suggestedOrder = [],
    ) {
        parent::__construct($message);
    }

    /**
     * The dependency keys that were not satisfied.
     *
     * @return array<int, string>
     *
     * @example
     * $exception->missingKeys(); // ['workspace_id']
     */
    public function missingKeys(): array
    {
        return array_values(array_filter(array_map(
            static fn (array $dependency): string => (string) ($dependency['key'] ?? ''),
            $this->missingDependencies,
        )));
    }
}
