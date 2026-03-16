<?php

declare(strict_types=1);

namespace Core\Mod\Agentic\Actions\Sprint;

use Core\Actions\Action;
use Core\Mod\Agentic\Models\Sprint;

/**
 * Get detailed information about a specific sprint.
 *
 * Usage:
 *   $sprint = GetSprint::run('sprint-1-abc123', 1);
 */
class GetSprint
{
    use Action;

    /**
     * @throws \InvalidArgumentException
     */
    public function handle(string $slug, int $workspaceId): Sprint
    {
        if ($slug === '') {
            throw new \InvalidArgumentException('slug is required');
        }

        $sprint = Sprint::with('issues')
            ->forWorkspace($workspaceId)
            ->where('slug', $slug)
            ->first();

        if (! $sprint) {
            throw new \InvalidArgumentException("Sprint not found: {$slug}");
        }

        return $sprint;
    }
}
