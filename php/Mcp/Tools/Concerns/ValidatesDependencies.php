<?php

// SPDX-License-Identifier: EUPL-1.2

declare(strict_types=1);

namespace Core\Mcp\Tools\Concerns;

use Core\Mcp\Dependencies\ToolDependency;
use Core\Mcp\Exceptions\MissingDependencyException;
use Core\Mod\Agentic\Mcp\Services\ToolDependencyService;

/**
 * Inline dependency validation for tools.
 *
 * AgentTool has used this trait since it was written, but the trait was never
 * present in this repository — it lives in dappcore/mcp, which is not a
 * dependency here. Every tool extending AgentTool was therefore fatal on load
 * ("Trait ... not found"), so none of the forty registered tool classes could
 * be constructed in any environment.
 *
 * Written against THIS repository's ToolDependencyService rather than copied
 * from upstream, because the two services share a name and not an API:
 *
 *   upstream trait calls        this service provides
 *   ------------------------    ------------------------------------------
 *   validateDependencies(...)   validateDependencies(mixed ...$arguments)
 *                               — variadic, so upstream's named arguments
 *                                 (sessionId:, toolName:, args:) bind to
 *                                 nothing at all
 *   checkDependencies(...)      canExecute($toolId, $context, $args, $session)
 *   getMissingDependencies(...) missing($toolId, $context, $args, $session)
 *   recordToolCall(...)         recordToolCall($sessionId, $toolId, $payload)
 *
 * A verbatim roll would have swapped a missing trait for a missing method.
 */
trait ValidatesDependencies
{
    /**
     * Dependencies this tool requires. Override to declare them.
     *
     * @return array<int, ToolDependency>
     *
     * @example
     * return [ToolDependency::contextExists('workspace_id')];
     */
    public function dependencies(): array
    {
        return [];
    }

    /**
     * Throw unless every declared dependency is satisfied.
     *
     * Positional, not named: the service takes mixed ...$arguments and parses
     * them by position and type.
     *
     * @throws MissingDependencyException
     *
     * @example
     * $this->validateDependencies(['session_id' => 'abc'], $args);
     */
    protected function validateDependencies(array $context = [], array $args = []): void
    {
        app(ToolDependencyService::class)->validateDependencies(
            $this->dependencySessionId($context),
            $this->name(),
            $context,
            $args,
        );
    }

    /**
     * Whether every declared dependency is satisfied, without throwing.
     *
     * @example
     * if (! $this->dependenciesMet($context, $args)) { ... }
     */
    protected function dependenciesMet(array $context = [], array $args = []): bool
    {
        return app(ToolDependencyService::class)->canExecute(
            $this->name(),
            $context,
            $args,
            $this->dependencySessionId($context),
        );
    }

    /**
     * The dependencies that are not satisfied.
     *
     * Returns the service's own row shape — arrays of
     * {tool, type, key, message} — rather than ToolDependency objects, because
     * that is what it produces.
     *
     * @return array<int, array{tool: string, type: string, key: string, message: string}>
     *
     * @example
     * $missing = $this->getMissingDependencies($context, $args);
     */
    protected function getMissingDependencies(array $context = [], array $args = []): array
    {
        return app(ToolDependencyService::class)->missing(
            $this->name(),
            $context,
            $args,
            $this->dependencySessionId($context),
        );
    }

    /**
     * Record this call so later tools can depend on it having happened.
     *
     * @example
     * $this->recordToolCall($context, $args);
     */
    protected function recordToolCall(array $context = [], array $args = []): void
    {
        app(ToolDependencyService::class)->recordToolCall(
            $this->dependencySessionId($context),
            $this->name(),
            $args,
        );
    }

    /**
     * Shape an unmet-dependency failure as a tool error response.
     *
     * @example
     * return $this->dependencyError($exception);
     */
    protected function dependencyError(MissingDependencyException $exception): array
    {
        return [
            'error' => 'dependency_not_met',
            'message' => $exception->getMessage(),
            'missing' => array_map(
                static fn (array $dependency): array => [
                    'type' => $dependency['type'] ?? 'unknown',
                    'key' => $dependency['key'] ?? '',
                    'description' => $dependency['message'] ?? '',
                ],
                $exception->missingDependencies,
            ),
            'suggested_order' => $exception->suggestedOrder,
        ];
    }

    /**
     * The session a dependency check is scoped to.
     */
    private function dependencySessionId(array $context): string
    {
        $sessionId = $context['session_id'] ?? null;

        return is_string($sessionId) && $sessionId !== '' ? $sessionId : 'anonymous';
    }
}
