<?php

// SPDX-License-Identifier: EUPL-1.2

declare(strict_types=1);

namespace Core\Mod\Agentic\Mcp\Resources\Agent;

use Core\Mod\Agentic\Models\AgentPlan;

/**
 * plans://{slug}/phases/{order} — one phase rendered as a task checklist.
 *
 * Not advertised in resources/list: the set is every phase of every plan, which
 * would swamp the listing. It is addressable directly, which is how the plan
 * document's own phase references are followed.
 *
 * @example
 * (new PhaseChecklistResource())->read('plans://ship-the-thing/phases/2');
 */
final class PhaseChecklistResource extends AgentResource
{
    public function uriTemplate(): string
    {
        return 'plans://{slug}/phases/{order}';
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

        [$slug, $order] = $parts;

        $plan = AgentPlan::where('slug', $slug)->first();
        if (! $plan) {
            return null;
        }

        $phase = $plan->agentPhases()->where('order', $order)->first();
        if (! $phase) {
            return null;
        }

        $markdown = "# Phase {$phase->order}: {$phase->name}\n\n";
        $markdown .= "**Status:** {$phase->getStatusIcon()} {$phase->status}\n\n";

        if ($phase->description) {
            $markdown .= "{$phase->description}\n\n";
        }

        $markdown .= "## Tasks\n\n";

        foreach ($phase->tasks ?? [] as $task) {
            // Tasks are stored either as a bare name or as a {name, status} map.
            $status = is_string($task) ? 'pending' : ($task['status'] ?? 'pending');
            $name = is_string($task) ? $task : ($task['name'] ?? 'Unknown');
            $icon = $status === 'completed' ? '✅' : '⬜';
            $markdown .= "- {$icon} {$name}\n";
        }

        return $markdown;
    }

    public function entries(): array
    {
        return [];
    }

    /**
     * @return array{0: string, 1: int}|null
     */
    private function partsFor(string $uri): ?array
    {
        if (! str_starts_with($uri, 'plans://')) {
            return null;
        }

        $parts = explode('/', substr($uri, strlen('plans://')));

        if (count($parts) !== 3 || $parts[1] !== 'phases' || $parts[0] === '') {
            return null;
        }

        // Numeric check before the cast: "phases/latest" is not phase 0.
        if (! is_numeric($parts[2])) {
            return null;
        }

        return [$parts[0], (int) $parts[2]];
    }
}
