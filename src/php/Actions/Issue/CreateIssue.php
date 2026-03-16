<?php

declare(strict_types=1);

namespace Core\Mod\Agentic\Actions\Issue;

use Core\Actions\Action;
use Core\Mod\Agentic\Models\Issue;
use Illuminate\Support\Str;

/**
 * Create a new issue.
 *
 * Usage:
 *   $issue = CreateIssue::run(['title' => 'Fix login bug', 'type' => 'bug'], 1);
 */
class CreateIssue
{
    use Action;

    /**
     * @param  array{title: string, slug?: string, description?: string, type?: string, priority?: string, labels?: array, assignee?: string, reporter?: string, sprint_id?: int, metadata?: array}  $data
     *
     * @throws \InvalidArgumentException
     */
    public function handle(array $data, int $workspaceId): Issue
    {
        $title = $data['title'] ?? null;
        if (! is_string($title) || $title === '' || mb_strlen($title) > 255) {
            throw new \InvalidArgumentException('title is required and must be a non-empty string (max 255 characters)');
        }

        $slug = $data['slug'] ?? null;
        if ($slug !== null) {
            if (! is_string($slug) || mb_strlen($slug) > 255) {
                throw new \InvalidArgumentException('slug must be a string (max 255 characters)');
            }
        } else {
            $slug = Str::slug($title).'-'.Str::random(6);
        }

        if (Issue::where('slug', $slug)->exists()) {
            throw new \InvalidArgumentException("Issue with slug '{$slug}' already exists");
        }

        $type = $data['type'] ?? Issue::TYPE_TASK;
        $validTypes = [Issue::TYPE_BUG, Issue::TYPE_FEATURE, Issue::TYPE_TASK, Issue::TYPE_IMPROVEMENT];
        if (! in_array($type, $validTypes, true)) {
            throw new \InvalidArgumentException(
                sprintf('type must be one of: %s', implode(', ', $validTypes))
            );
        }

        $priority = $data['priority'] ?? Issue::PRIORITY_NORMAL;
        $validPriorities = [Issue::PRIORITY_LOW, Issue::PRIORITY_NORMAL, Issue::PRIORITY_HIGH, Issue::PRIORITY_URGENT];
        if (! in_array($priority, $validPriorities, true)) {
            throw new \InvalidArgumentException(
                sprintf('priority must be one of: %s', implode(', ', $validPriorities))
            );
        }

        $issue = Issue::create([
            'workspace_id' => $workspaceId,
            'sprint_id' => $data['sprint_id'] ?? null,
            'slug' => $slug,
            'title' => $title,
            'description' => $data['description'] ?? null,
            'type' => $type,
            'status' => Issue::STATUS_OPEN,
            'priority' => $priority,
            'labels' => $data['labels'] ?? [],
            'assignee' => $data['assignee'] ?? null,
            'reporter' => $data['reporter'] ?? null,
            'metadata' => $data['metadata'] ?? [],
        ]);

        return $issue->load('sprint');
    }
}
