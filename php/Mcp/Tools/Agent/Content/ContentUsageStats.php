<?php

declare(strict_types=1);

namespace Core\Mod\Agentic\Mcp\Tools\Agent\Content;

use Core\Mod\Agentic\Mcp\Tools\Agent\AgentTool;
use Mod\Content\Models\AIUsage;

/**
 * Get AI usage statistics for content generation.
 *
 * Returns token counts and cost estimates by provider and purpose.
 */
class ContentUsageStats extends AgentTool
{
    protected string $category = 'content';

    protected array $scopes = ['read'];

    public function name(): string
    {
        return 'content_usage_stats';
    }

    public function description(): string
    {
        return 'Get AI usage statistics (tokens, costs) for content generation';
    }

    public function inputSchema(): array
    {
        return [
            'type' => 'object',
            'properties' => [
                'period' => [
                    'type' => 'string',
                    'description' => 'Time period for stats',
                    'enum' => ['day', 'week', 'month', 'year'],
                ],
            ],
        ];
    }

    public function handle(array $args, array $context = []): array
    {
        try {
            $period = $this->optionalEnum($args, 'period', ['day', 'week', 'month', 'year'], 'month');
        } catch (\InvalidArgumentException $e) {
            return $this->error($e->getMessage());
        }

        // Use workspace_id from context if available (null returns system-wide stats)
        $workspaceId = $context['workspace_id'] ?? null;

        $stats = AIUsage::statsForWorkspace($workspaceId, $period);

        return $this->success([
            'period' => $period,
            'total_requests' => $stats['total_requests'],
            'total_input_tokens' => (int) $stats['total_input_tokens'],
            'total_output_tokens' => (int) $stats['total_output_tokens'],
            'total_cost' => number_format((float) $stats['total_cost'], 4),
            'by_provider' => $stats['by_provider'],
            'by_purpose' => $stats['by_purpose'],
        ]);
    }
}
