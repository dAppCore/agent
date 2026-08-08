<?php

// SPDX-License-Identifier: EUPL-1.2

declare(strict_types=1);

namespace Core\Mod\Agentic\Mcp\Resources\Agent;

use Core\Mod\Agentic\Models\AgentPlan;

/**
 * plans://{slug}/state/{key} — one workspace state value, formatted.
 *
 * Not advertised in resources/list for the same reason as the phase checklist:
 * the set is unbounded and per-plan. Addressable directly.
 *
 * @example
 * (new StateValueResource())->read('plans://ship-the-thing/state/build_id');
 */
final class StateValueResource extends AgentResource
{
    public function uriTemplate(): string
    {
        return 'plans://{slug}/state/{key}';
    }

    public function matches(string $uri): bool
    {
        return $this->partsFor($uri) !== null;
    }

    public function read(string $uri): ?string
    {
        $parts = $this->partsFor($uri);
        if ($parts === null) {
            return null;
        }

        [$slug, $key] = $parts;

        $plan = AgentPlan::where('slug', $slug)->first();
        if (! $plan) {
            return null;
        }

        $state = $plan->states()->where('key', $key)->first();

        return $state?->getFormattedValue();
    }

    public function entries(): array
    {
        return [];
    }

    /**
     * @return array{0: string, 1: string}|null
     */
    private function partsFor(string $uri): ?array
    {
        if (! str_starts_with($uri, 'plans://')) {
            return null;
        }

        $parts = explode('/', substr($uri, strlen('plans://')));

        if (count($parts) !== 3 || $parts[1] !== 'state' || $parts[0] === '' || $parts[2] === '') {
            return null;
        }

        return [$parts[0], $parts[2]];
    }
}
