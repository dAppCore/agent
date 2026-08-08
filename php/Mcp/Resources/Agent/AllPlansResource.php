<?php

// SPDX-License-Identifier: EUPL-1.2

declare(strict_types=1);

namespace Core\Mod\Agentic\Mcp\Resources\Agent;

use Core\Mod\Agentic\Models\AgentPlan;

/**
 * plans://all — a markdown overview of every non-archived plan, grouped by
 * status with each plan's completion percentage.
 *
 * @example
 * (new AllPlansResource())->read('plans://all');
 */
final class AllPlansResource extends AgentResource
{
    public const URI = 'plans://all';

    public function uriTemplate(): string
    {
        return self::URI;
    }

    public function matches(string $uri): bool
    {
        return $uri === self::URI;
    }

    public function read(string $uri): ?string
    {
        if (! $this->matches($uri)) {
            return null;
        }

        $plans = AgentPlan::with('agentPhases')
            ->notArchived()
            ->orderBy('updated_at', 'desc')
            ->get();

        $markdown = "# Work Plans\n\n";
        $markdown .= '**Total:** '.$plans->count()." plan(s)\n\n";

        foreach ($plans->groupBy('status') as $status => $group) {
            $markdown .= '## '.ucfirst((string) $status).' ('.$group->count().")\n\n";

            foreach ($group as $plan) {
                $progress = $plan->getProgress();
                $markdown .= "- **[{$plan->slug}]** {$plan->title} - {$progress['percentage']}%\n";
            }

            $markdown .= "\n";
        }

        return $markdown;
    }

    public function entries(): array
    {
        return [
            $this->entry(self::URI, 'All Plans Overview', 'Overview of all work plans and their status'),
        ];
    }
}
