<?php

// SPDX-License-Identifier: EUPL-1.2

declare(strict_types=1);

namespace Core\Mod\Agentic\Mcp\Resources\Agent;

use Core\Mod\Agentic\Models\AgentPlan;

/**
 * plans://{slug} — one plan rendered as its markdown document.
 *
 * Also the resource that makes resources/list dynamic: it advertises an entry
 * per non-archived plan, so a client sees the plans that exist rather than a
 * template it has to guess slugs for.
 *
 * @example
 * (new PlanDocumentResource())->read('plans://ship-the-thing');
 */
final class PlanDocumentResource extends AgentResource
{
    public function uriTemplate(): string
    {
        return 'plans://{slug}';
    }

    public function matches(string $uri): bool
    {
        return $this->slugFor($uri) !== null;
    }

    public function read(string $uri): ?string
    {
        $slug = $this->slugFor($uri);
        if ($slug === null) {
            return null;
        }

        $plan = AgentPlan::with('agentPhases')->where('slug', $slug)->first();

        return $plan?->toMarkdown();
    }

    public function entries(): array
    {
        return AgentPlan::notArchived()
            ->get()
            ->map(fn (AgentPlan $plan): array => $this->entry(
                "plans://{$plan->slug}",
                (string) $plan->title,
                "Work plan: {$plan->title}",
            ))
            ->all();
    }

    /**
     * Extract the slug from a bare plans://{slug} URI.
     *
     * Returns null for plans://all and for the deeper phases/state URIs, which
     * belong to the other resources — the registry asks each resource in turn,
     * so each has to decline anything that is not precisely its own shape.
     */
    private function slugFor(string $uri): ?string
    {
        if (! str_starts_with($uri, 'plans://') || $uri === AllPlansResource::URI) {
            return null;
        }

        $path = substr($uri, strlen('plans://'));
        if ($path === '' || str_contains($path, '/')) {
            return null;
        }

        return $path;
    }
}
