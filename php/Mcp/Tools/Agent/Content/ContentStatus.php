<?php

declare(strict_types=1);

namespace Core\Mod\Agentic\Mcp\Tools\Agent\Content;

use Core\Mod\Agentic\Mcp\Tools\Agent\AgentTool;
use Core\Mod\Content\Models\ContentBrief;
use Core\Mod\Content\Services\AIGatewayService;

/**
 * Get content generation pipeline status.
 *
 * Returns AI provider availability and brief counts by status.
 */
class ContentStatus extends AgentTool
{
    protected string $category = 'content';

    protected array $scopes = ['read'];

    public function name(): string
    {
        return 'content_status';
    }

    public function description(): string
    {
        return 'Get content generation pipeline status (AI provider availability, brief counts)';
    }

    public function inputSchema(): array
    {
        return [
            'type' => 'object',
            'properties' => (object) [],
        ];
    }

    public function handle(array $args, array $context = []): array
    {
        $gateway = app(AIGatewayService::class);

        return $this->success([
            'providers' => [
                'gemini' => $gateway->isGeminiAvailable(),
                'claude' => $gateway->isClaudeAvailable(),
            ],
            'pipeline_available' => $gateway->isAvailable(),
            'briefs' => [
                'pending' => ContentBrief::pending()->count(),
                'queued' => ContentBrief::where('status', ContentBrief::STATUS_QUEUED)->count(),
                'generating' => ContentBrief::where('status', ContentBrief::STATUS_GENERATING)->count(),
                'review' => ContentBrief::needsReview()->count(),
                'published' => ContentBrief::where('status', ContentBrief::STATUS_PUBLISHED)->count(),
                'failed' => ContentBrief::where('status', ContentBrief::STATUS_FAILED)->count(),
            ],
        ]);
    }
}
