<?php

declare(strict_types=1);

namespace Core\Mod\Agentic\Actions\Issue;

use Core\Actions\Action;
use Core\Mod\Agentic\Models\Issue;

/**
 * Get detailed information about a specific issue.
 *
 * Usage:
 *   $issue = GetIssue::run('fix-login-bug-abc123', 1);
 */
class GetIssue
{
    use Action;

    /**
     * @throws \InvalidArgumentException
     */
    public function handle(string $slug, int $workspaceId): Issue
    {
        if ($slug === '') {
            throw new \InvalidArgumentException('slug is required');
        }

        $issue = Issue::with(['sprint', 'comments'])
            ->forWorkspace($workspaceId)
            ->where('slug', $slug)
            ->first();

        if (! $issue) {
            throw new \InvalidArgumentException("Issue not found: {$slug}");
        }

        return $issue;
    }
}
