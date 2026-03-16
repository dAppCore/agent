<?php

declare(strict_types=1);

namespace Core\Mod\Agentic\Actions\Sprint;

use Core\Actions\Action;
use Core\Mod\Agentic\Models\Sprint;

/**
 * Archive (cancel) a sprint.
 *
 * Usage:
 *   $sprint = ArchiveSprint::run('sprint-1', 1, 'Scope changed');
 */
class ArchiveSprint
{
    use Action;

    /**
     * @throws \InvalidArgumentException
     */
    public function handle(string $slug, int $workspaceId, ?string $reason = null): Sprint
    {
        if ($slug === '') {
            throw new \InvalidArgumentException('slug is required');
        }

        $sprint = Sprint::forWorkspace($workspaceId)
            ->where('slug', $slug)
            ->first();

        if (! $sprint) {
            throw new \InvalidArgumentException("Sprint not found: {$slug}");
        }

        $sprint->cancel($reason);

        return $sprint->fresh();
    }
}
