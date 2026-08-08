<?php

declare(strict_types=1);

namespace Core\Mod\Agentic\Mcp\Tools\Agent\Content;

use Core\Mod\Agentic\Mcp\Tools\Agent\AgentTool;
use Core\Mod\Content\Enums\BriefContentType;
use Core\Mod\Content\Models\ContentBrief;

/**
 * Get details of a specific content brief including generated content.
 */
class ContentBriefGet extends AgentTool
{
    protected string $category = 'content';

    protected array $scopes = ['read'];

    public function name(): string
    {
        return 'content_brief_get';
    }

    public function description(): string
    {
        return 'Get details of a specific content brief including generated content';
    }

    public function inputSchema(): array
    {
        return [
            'type' => 'object',
            'properties' => [
                'id' => [
                    'type' => 'integer',
                    'description' => 'Brief ID',
                ],
            ],
            'required' => ['id'],
        ];
    }

    public function handle(array $args, array $context = []): array
    {
        try {
            $id = $this->requireInt($args, 'id', 1);
        } catch (\InvalidArgumentException $e) {
            return $this->error($e->getMessage());
        }

        $brief = ContentBrief::find($id);

        if (! $brief) {
            return $this->error("Brief not found: {$id}");
        }

        // Optional workspace scoping for multi-tenant security
        if (! empty($context['workspace_id']) && $brief->workspace_id !== $context['workspace_id']) {
            return $this->error('Access denied: brief belongs to a different workspace');
        }

        return $this->success([
            'brief' => [
                'id' => $brief->id,
                'title' => $brief->title,
                'slug' => $brief->slug,
                'status' => $brief->status,
                'content_type' => $brief->content_type instanceof BriefContentType
                    ? $brief->content_type->value
                    : $brief->content_type,
                'service' => $brief->service,
                'description' => $brief->description,
                'keywords' => $brief->keywords,
                'target_word_count' => $brief->target_word_count,
                'difficulty' => $brief->difficulty,
                'draft_output' => $brief->draft_output,
                'refined_output' => $brief->refined_output,
                'final_content' => $brief->final_content,
                'error_message' => $brief->error_message,
                'generation_log' => $brief->generation_log,
                'metadata' => $brief->metadata,
                'total_cost' => $brief->total_cost,
                'created_at' => $brief->created_at->toIso8601String(),
                'updated_at' => $brief->updated_at->toIso8601String(),
                'generated_at' => $brief->generated_at?->toIso8601String(),
                'refined_at' => $brief->refined_at?->toIso8601String(),
                'published_at' => $brief->published_at?->toIso8601String(),
            ],
        ]);
    }
}
