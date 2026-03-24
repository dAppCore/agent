<?php

declare(strict_types=1);

namespace Core\Mod\Agentic\Mcp\Tools\Agent\Content;

use Core\Mod\Agentic\Mcp\Tools\Agent\AgentTool;
use Mod\Content\Enums\BriefContentType;
use Mod\Content\Models\ContentBrief;

/**
 * List content briefs with optional status filter.
 */
class ContentBriefList extends AgentTool
{
    protected string $category = 'content';

    protected array $scopes = ['read'];

    public function name(): string
    {
        return 'content_brief_list';
    }

    public function description(): string
    {
        return 'List content briefs with optional status filter';
    }

    public function inputSchema(): array
    {
        return [
            'type' => 'object',
            'properties' => [
                'status' => [
                    'type' => 'string',
                    'description' => 'Filter by status',
                    'enum' => ['pending', 'queued', 'generating', 'review', 'published', 'failed'],
                ],
                'limit' => [
                    'type' => 'integer',
                    'description' => 'Maximum results (default: 20)',
                ],
            ],
        ];
    }

    public function handle(array $args, array $context = []): array
    {
        try {
            $limit = $this->optionalInt($args, 'limit', 20, 1, 100);
            $status = $this->optionalEnum($args, 'status', [
                'pending', 'queued', 'generating', 'review', 'published', 'failed',
            ]);
        } catch (\InvalidArgumentException $e) {
            return $this->error($e->getMessage());
        }

        $query = ContentBrief::query()->orderBy('created_at', 'desc');

        // Scope to workspace if provided
        if (! empty($context['workspace_id'])) {
            $query->where('workspace_id', $context['workspace_id']);
        }

        if ($status) {
            $query->where('status', $status);
        }

        $briefs = $query->limit($limit)->get();

        return $this->success([
            'briefs' => $briefs->map(fn ($brief) => [
                'id' => $brief->id,
                'title' => $brief->title,
                'status' => $brief->status,
                'content_type' => $brief->content_type instanceof BriefContentType
                    ? $brief->content_type->value
                    : $brief->content_type,
                'service' => $brief->service,
                'created_at' => $brief->created_at->toIso8601String(),
            ])->all(),
            'total' => $briefs->count(),
        ]);
    }
}
