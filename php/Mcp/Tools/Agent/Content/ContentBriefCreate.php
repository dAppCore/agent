<?php

declare(strict_types=1);

namespace Core\Mod\Agentic\Mcp\Tools\Agent\Content;

use Core\Mod\Agentic\Mcp\Tools\Agent\AgentTool;
use Core\Mod\Agentic\Models\AgentPlan;
use Core\Mod\Content\Enums\BriefContentType;
use Core\Mod\Content\Models\ContentBrief;
use Illuminate\Support\Str;

/**
 * Create a content brief for AI generation.
 *
 * Briefs can be linked to an existing plan for workflow tracking.
 */
class ContentBriefCreate extends AgentTool
{
    protected string $category = 'content';

    protected array $scopes = ['write'];

    public function name(): string
    {
        return 'content_brief_create';
    }

    public function description(): string
    {
        return 'Create a content brief for AI generation';
    }

    public function inputSchema(): array
    {
        return [
            'type' => 'object',
            'properties' => [
                'title' => [
                    'type' => 'string',
                    'description' => 'Content title',
                ],
                'content_type' => [
                    'type' => 'string',
                    'description' => 'Type of content',
                    'enum' => BriefContentType::values(),
                ],
                'service' => [
                    'type' => 'string',
                    'description' => 'Service context (e.g., BioHost, QRHost)',
                ],
                'keywords' => [
                    'type' => 'array',
                    'description' => 'SEO keywords to include',
                    'items' => ['type' => 'string'],
                ],
                'target_word_count' => [
                    'type' => 'integer',
                    'description' => 'Target word count (default: 800)',
                ],
                'description' => [
                    'type' => 'string',
                    'description' => 'Brief description of what to write about',
                ],
                'difficulty' => [
                    'type' => 'string',
                    'description' => 'Target audience level',
                    'enum' => ['beginner', 'intermediate', 'advanced'],
                ],
                'plan_slug' => [
                    'type' => 'string',
                    'description' => 'Link to an existing plan',
                ],
            ],
            'required' => ['title', 'content_type'],
        ];
    }

    public function handle(array $args, array $context = []): array
    {
        try {
            $title = $this->requireString($args, 'title', 255);
            $contentType = $this->requireEnum($args, 'content_type', BriefContentType::values());
        } catch (\InvalidArgumentException $e) {
            return $this->error($e->getMessage());
        }

        $plan = null;
        if (! empty($args['plan_slug'])) {
            $plan = AgentPlan::where('slug', $args['plan_slug'])->first();
            if (! $plan) {
                return $this->error("Plan not found: {$args['plan_slug']}");
            }
        }

        // Determine workspace_id from context
        $workspaceId = $context['workspace_id'] ?? null;

        $brief = ContentBrief::create([
            'workspace_id' => $workspaceId,
            'title' => $title,
            'slug' => Str::slug($title).'-'.Str::random(6),
            'content_type' => $contentType,
            'service' => $args['service'] ?? null,
            'description' => $args['description'] ?? null,
            'keywords' => $args['keywords'] ?? null,
            'target_word_count' => $args['target_word_count'] ?? 800,
            'difficulty' => $args['difficulty'] ?? null,
            'status' => ContentBrief::STATUS_PENDING,
            'metadata' => $plan ? [
                'plan_id' => $plan->id,
                'plan_slug' => $plan->slug,
            ] : null,
        ]);

        return $this->success([
            'brief' => [
                'id' => $brief->id,
                'title' => $brief->title,
                'slug' => $brief->slug,
                'status' => $brief->status,
                'content_type' => $brief->content_type instanceof BriefContentType
                    ? $brief->content_type->value
                    : $brief->content_type,
            ],
        ]);
    }
}
