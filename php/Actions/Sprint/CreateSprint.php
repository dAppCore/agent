<?php

declare(strict_types=1);

namespace Core\Mod\Agentic\Actions\Sprint;

use Core\Actions\Action;
use Core\Mod\Agentic\Models\Sprint;
use Illuminate\Support\Str;

/**
 * Create a new sprint.
 *
 * Usage:
 *   $sprint = CreateSprint::run(['title' => 'Sprint 1', 'goal' => 'MVP launch'], 1);
 */
class CreateSprint
{
    use Action;

    /**
     * @param  array{title: string, slug?: string, description?: string, goal?: string, metadata?: array}  $data
     *
     * @throws \InvalidArgumentException
     */
    public function handle(array $data, int $workspaceId): Sprint
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

        if (Sprint::where('slug', $slug)->exists()) {
            throw new \InvalidArgumentException("Sprint with slug '{$slug}' already exists");
        }

        return Sprint::create([
            'workspace_id' => $workspaceId,
            'slug' => $slug,
            'title' => $title,
            'description' => $data['description'] ?? null,
            'goal' => $data['goal'] ?? null,
            'status' => Sprint::STATUS_PLANNING,
            'metadata' => $data['metadata'] ?? [],
        ]);
    }
}
