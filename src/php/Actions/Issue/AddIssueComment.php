<?php

declare(strict_types=1);

namespace Core\Mod\Agentic\Actions\Issue;

use Core\Actions\Action;
use Core\Mod\Agentic\Models\Issue;
use Core\Mod\Agentic\Models\IssueComment;

/**
 * Add a comment to an issue.
 *
 * Usage:
 *   $comment = AddIssueComment::run('fix-login-bug', 1, 'claude', 'Investigating root cause.');
 */
class AddIssueComment
{
    use Action;

    /**
     * @throws \InvalidArgumentException
     */
    public function handle(string $slug, int $workspaceId, string $author, string $body, ?array $metadata = null): IssueComment
    {
        if ($slug === '') {
            throw new \InvalidArgumentException('slug is required');
        }

        if ($author === '') {
            throw new \InvalidArgumentException('author is required');
        }

        if ($body === '') {
            throw new \InvalidArgumentException('body is required');
        }

        $issue = Issue::forWorkspace($workspaceId)
            ->where('slug', $slug)
            ->first();

        if (! $issue) {
            throw new \InvalidArgumentException("Issue not found: {$slug}");
        }

        return IssueComment::create([
            'issue_id' => $issue->id,
            'author' => $author,
            'body' => $body,
            'metadata' => $metadata,
        ]);
    }
}
