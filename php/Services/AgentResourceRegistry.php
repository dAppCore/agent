<?php

// SPDX-License-Identifier: EUPL-1.2

declare(strict_types=1);

namespace Core\Mod\Agentic\Services;

use Core\Mod\Agentic\Mcp\Resources\Agent\AgentResource;

/**
 * Registry for MCP Agent Server resources.
 *
 * The counterpart to AgentToolRegistry: Boot registers the resource classes,
 * the stdio command asks this for its resources/list payload and routes
 * resources/read through it. Keeping the routing here rather than in the
 * command is what stopped this surface being a 2000-line match statement in
 * the first place.
 */
final class AgentResourceRegistry
{
    /**
     * @var array<int, AgentResource>
     */
    private array $resources = [];

    /**
     * Register one resource.
     *
     * @example
     * $registry->register(new AllPlansResource());
     */
    public function register(AgentResource $resource): self
    {
        $this->resources[] = $resource;

        return $this;
    }

    /**
     * Register several resources at once.
     *
     * @param  iterable<AgentResource>  $resources
     *
     * @example
     * $registry->registerMany([new AllPlansResource(), new PlanDocumentResource()]);
     */
    public function registerMany(iterable $resources): self
    {
        foreach ($resources as $resource) {
            $this->register($resource);
        }

        return $this;
    }

    /**
     * Every entry resources/list should advertise.
     *
     * Resources contribute zero or more entries, so a per-plan resource can
     * enumerate what actually exists while the unbounded per-key ones (phase
     * checklists, state values) advertise nothing and stay directly addressable.
     *
     * @return array<int, array{uri: string, name: string, description: string, mimeType: string}>
     *
     * @example
     * $registry->entries();
     */
    public function entries(): array
    {
        $entries = [];

        foreach ($this->resources as $resource) {
            foreach ($resource->entries() as $entry) {
                $entries[] = $entry;
            }
        }

        return $entries;
    }

    /**
     * Resolve the resource that serves a URI.
     *
     * @example
     * $registry->resolve('plans://all');
     */
    public function resolve(string $uri): ?AgentResource
    {
        foreach ($this->resources as $resource) {
            if ($resource->matches($uri)) {
                return $resource;
            }
        }

        return null;
    }

    /**
     * Read a URI, or null when nothing serves it.
     *
     * Null covers both "no resource matches this shape" and "the shape matched
     * but names something that does not exist", so the command answers a proper
     * JSON-RPC error either way rather than handing back a document that reads
     * like content but says "Plan not found".
     *
     * @return array{uri: string, mimeType: string, text: string}|null
     *
     * @example
     * $registry->read('plans://ship-the-thing');
     */
    public function read(string $uri): ?array
    {
        $resource = $this->resolve($uri);
        if (! $resource) {
            return null;
        }

        $text = $resource->read($uri);
        if ($text === null) {
            return null;
        }

        return [
            'uri' => $uri,
            'mimeType' => $resource->mimeType(),
            'text' => $text,
        ];
    }
}
