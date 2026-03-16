<?php

declare(strict_types=1);

namespace Core\Mod\Agentic\Actions\Issue;

use Core\Actions\Action;
use Core\Mod\Agentic\Models\Issue;

/**
 * Archive an issue.
 *
 * Usage:
 *   $issue = ArchiveIssue::run('fix-login-bug', 1, 'Duplicate of #42');
 */
class ArchiveIssue
{
    use Action;

    /**
     * @throws \InvalidArgumentException
     */
    public function handle(string $slug, int $workspaceId, ?string $reason = null): Issue
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

        $issue->archive($reason);

        return $issue->fresh();
    }
}
