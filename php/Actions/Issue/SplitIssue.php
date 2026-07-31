<?php

// SPDX-License-Identifier: EUPL-1.2

declare(strict_types=1);

namespace Core\Mod\Agentic\Actions\Issue;

use Core\Actions\Action;
use Core\Mod\Agentic\Models\Issue;
use Illuminate\Support\Collection;
use Illuminate\Support\Facades\DB;

/**
 * Split an issue into child issues.
 *
 * This is the onboarding move: something arrives too big to work — a scan
 * result, a customer ask, a months-long programme — and is broken into pieces
 * that each fit one person, one discipline, one sitting. Children inherit the
 * parent's workspace, sprint and repository, so a chunk needs only a title.
 *
 * Usage:
 *   $epic = CreateIssue::run(['title' => 'Retire go.work', 'type' => 'epic'], 1);
 *   $children = SplitIssue::run($epic, [
 *       ['title' => 'Map the dependency layers', 'discipline' => 'planning'],
 *       ['title' => 'Tag the L0 modules',        'discipline' => 'code', 'priority' => 'high'],
 *       ['title' => 'Rewrite the release guide', 'discipline' => 'content'],
 *   ]);
 */
class SplitIssue
{
    use Action;

    /**
     * @param  Issue|string  $parent  the parent issue, or its slug
     * @param  list<array{title: string, description?: string, type?: string, discipline?: string, priority?: string, labels?: array, assignee?: string, repository?: string, reporter?: string, metadata?: array}>  $chunks
     * @return Collection<int, Issue>
     *
     * @throws \InvalidArgumentException
     */
    public function handle(Issue|string $parent, array $chunks)
    {
        if (is_string($parent)) {
            $slug = $parent;
            $parent = Issue::where('slug', $slug)->first();

            if ($parent === null) {
                throw new \InvalidArgumentException("Issue with slug '{$slug}' does not exist");
            }
        }

        if ($parent->workspace_id === null) {
            throw new \InvalidArgumentException("Cannot split issue '{$parent->slug}': it has no workspace");
        }

        if ($chunks === []) {
            throw new \InvalidArgumentException('At least one chunk is required');
        }

        foreach ($chunks as $index => $chunk) {
            if (! isset($chunk['title']) || ! is_string($chunk['title']) || trim($chunk['title']) === '') {
                throw new \InvalidArgumentException("Chunk at index {$index} is missing a title");
            }
        }

        // All-or-nothing: a half-applied split leaves an epic whose progress
        // reads as further along than the work actually is, which is worse
        // than the split having plainly failed.
        return DB::transaction(function () use ($parent, $chunks) {
            $created = collect();

            foreach ($chunks as $chunk) {
                $chunk['parent_id'] = $parent->id;

                // An epic's children default to ordinary tasks; nothing stops a
                // caller nesting a further epic, but it should be deliberate.
                $chunk['type'] ??= Issue::TYPE_TASK;

                $created->push(CreateIssue::run($chunk, $parent->workspace_id));
            }

            return $created;
        });
    }
}
