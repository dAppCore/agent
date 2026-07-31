<?php

declare(strict_types=1);

namespace Core\Mod\Agentic\Actions\Issue;

use Core\Actions\Action;
use Core\Mod\Agentic\Models\Issue;

/**
 * Update an issue's fields.
 *
 * Usage:
 *   $issue = UpdateIssue::run('fix-login-bug', ['status' => 'in_progress'], 1);
 */
class UpdateIssue
{
    use Action;

    /**
     * @param  array{status?: string, priority?: string, type?: string, title?: string, description?: string, assignee?: string, sprint_id?: int|null, labels?: array}  $data
     *
     * @throws \InvalidArgumentException
     */
    public function handle(string $slug, array $data, int $workspaceId): Issue
    {
        if ($slug === '') {
            throw new \InvalidArgumentException('slug is required');
        }

        $issue = Issue::forWorkspace($workspaceId)
            ->where('slug', $slug)
            ->first();

        if (! $issue) {
            throw new \InvalidArgumentException("Issue not found: {$slug}");
        }

        if (isset($data['status'])) {
            $valid = [Issue::STATUS_OPEN, Issue::STATUS_IN_PROGRESS, Issue::STATUS_REVIEW, Issue::STATUS_CLOSED];
            if (! in_array($data['status'], $valid, true)) {
                throw new \InvalidArgumentException(
                    sprintf('status must be one of: %s', implode(', ', $valid))
                );
            }

            if ($data['status'] === Issue::STATUS_CLOSED) {
                // Route through the model so closing by status update is held
                // to the same rule as Issue::close() — an epic does not close
                // out from under its unfinished children.
                $open = $issue->openChildren();

                if ($open->isNotEmpty()) {
                    throw new \RuntimeException(sprintf(
                        "Cannot close issue '%s': %d child issue(s) still open (%s). Close them first, or archive.",
                        $issue->slug,
                        $open->count(),
                        $open->pluck('slug')->implode(', ')
                    ));
                }

                $data['closed_at'] = now();
            } elseif ($issue->status === Issue::STATUS_CLOSED) {
                $data['closed_at'] = null;
            }
        }

        if (isset($data['priority'])) {
            $valid = [Issue::PRIORITY_LOW, Issue::PRIORITY_NORMAL, Issue::PRIORITY_HIGH, Issue::PRIORITY_URGENT];
            if (! in_array($data['priority'], $valid, true)) {
                throw new \InvalidArgumentException(
                    sprintf('priority must be one of: %s', implode(', ', $valid))
                );
            }
        }

        if (isset($data['type'])) {
            $valid = [Issue::TYPE_BUG, Issue::TYPE_FEATURE, Issue::TYPE_TASK, Issue::TYPE_IMPROVEMENT, Issue::TYPE_EPIC];
            if (! in_array($data['type'], $valid, true)) {
                throw new \InvalidArgumentException(
                    sprintf('type must be one of: %s', implode(', ', $valid))
                );
            }
        }

        if (isset($data['discipline']) && ! in_array($data['discipline'], Issue::disciplines(), true)) {
            throw new \InvalidArgumentException(
                sprintf('discipline must be one of: %s', implode(', ', Issue::disciplines()))
            );
        }

        $issue->update($data);

        return $issue->fresh()->load('sprint', 'parent');
    }
}
