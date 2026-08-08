<?php

// SPDX-License-Identifier: EUPL-1.2

declare(strict_types=1);

namespace Core\Mod\Agentic\Mcp\Resources\Agent;

/**
 * Base class for MCP Agent Server resources.
 *
 * Mirrors the shape of Mcp\Tools\Agent\AgentTool: one class per resource,
 * registered into a registry, rather than a handler method on a monolithic
 * command. These carry the plans/phases/sessions/state surface ported out of
 * dappcore/mcp, which owned it only because that command was filed in the
 * shared package — the data is entirely agent-domain.
 *
 * A resource answers two questions: what should `resources/list` advertise
 * (entries(), which may be dynamic), and what does `resources/read` return for
 * a given URI (matches() then read()).
 */
abstract class AgentResource
{
    /**
     * The URI template this resource answers, for documentation and errors.
     *
     * @example
     * $resource->uriTemplate(); // 'plans://{slug}/phases/{order}'
     */
    abstract public function uriTemplate(): string;

    /**
     * Whether this resource can serve the given URI.
     *
     * @example
     * $resource->matches('plans://all'); // true
     */
    abstract public function matches(string $uri): bool;

    /**
     * Render the resource body for a URI this resource matches.
     *
     * Returns null when the URI matches the shape but names something that does
     * not exist, so the caller can answer "not found" rather than an empty
     * document that reads like real content.
     *
     * @example
     * $resource->read('plans://all');
     */
    abstract public function read(string $uri): ?string;

    /**
     * Entries this resource contributes to resources/list.
     *
     * Returns a list so one resource can advertise many URIs — the plan
     * document resource enumerates every non-archived plan.
     *
     * @return array<int, array{uri: string, name: string, description: string, mimeType: string}>
     *
     * @example
     * $resource->entries(); // [['uri' => 'plans://all', ...]]
     */
    abstract public function entries(): array;

    /**
     * The MIME type this resource renders. Markdown throughout.
     */
    public function mimeType(): string
    {
        return 'text/markdown';
    }

    /**
     * Build a single list entry, so subclasses do not repeat the shape.
     *
     * @return array{uri: string, name: string, description: string, mimeType: string}
     */
    protected function entry(string $uri, string $name, string $description): array
    {
        return [
            'uri' => $uri,
            'name' => $name,
            'description' => $description,
            'mimeType' => $this->mimeType(),
        ];
    }
}
