<?php

declare(strict_types=1);

namespace Core\Mod\Agentic\Actions\Sprint;

use Core\Actions\Action;
use Core\Mod\Agentic\Models\Sprint;

/**
 * Update a sprint's fields.
 *
 * Usage:
 *   $sprint = UpdateSprint::run('sprint-1', ['status' => 'active'], 1);
 */
class UpdateSprint
{
    use Action;

    /**
     * @param  array{status?: string, title?: string, description?: string, goal?: string}  $data
     *
     * @throws \InvalidArgumentException
     */
    public function handle(string $slug, array $data, int $workspaceId): Sprint
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

        if (isset($data['status'])) {
            $valid = [Sprint::STATUS_PLANNING, Sprint::STATUS_ACTIVE, Sprint::STATUS_COMPLETED, Sprint::STATUS_CANCELLED];
            if (! in_array($data['status'], $valid, true)) {
                throw new \InvalidArgumentException(
                    sprintf('status must be one of: %s', implode(', ', $valid))
                );
            }

            if ($data['status'] === Sprint::STATUS_ACTIVE && ! $sprint->started_at) {
                $data['started_at'] = now();
            }

            if (in_array($data['status'], [Sprint::STATUS_COMPLETED, Sprint::STATUS_CANCELLED], true)) {
                $data['ended_at'] = now();
            }
        }

        $sprint->update($data);

        return $sprint->fresh()->load('issues');
    }
}
