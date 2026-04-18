<?php

declare(strict_types=1);

namespace Core\Mod\Agentic\Mcp\Tools\Agent\Plan;

use Core\Mcp\Dependencies\ToolDependency;
use Core\Mod\Agentic\Actions\Plan\GetPlan;
use Core\Mod\Agentic\Mcp\Tools\Agent\AgentTool;

/**
 * Get detailed information about a specific plan.
 */
class PlanGet extends AgentTool
{
    protected string $category = 'plan';

    protected array $scopes = ['read'];

    /**
     * Get the dependencies for this tool.
     *
     * Workspace context is required to ensure tenant isolation.
     *
     * @return array<ToolDependency>
     */
    public function dependencies(): array
    {
        return [
            ToolDependency::contextExists('workspace_id', 'Workspace context required for plan operations'),
        ];
    }

    public function name(): string
    {
        return 'plan_get';
    }

    public function description(): string
    {
        return 'Get detailed information about a specific plan';
    }

    public function inputSchema(): array
    {
        return [
            'type' => 'object',
            'properties' => [
                'slug' => [
                    'type' => 'string',
                    'description' => 'Plan slug identifier',
                ],
                'format' => [
                    'type' => 'string',
                    'description' => 'Output format: json or markdown',
                    'enum' => ['json', 'markdown'],
                ],
            ],
            'required' => ['slug'],
        ];
    }

    public function handle(array $args, array $context = []): array
    {
        $workspaceId = $context['workspace_id'] ?? null;
        if ($workspaceId === null) {
            return $this->error('workspace_id is required. Ensure you have authenticated with a valid API key and started a session. See: https://host.uk.com/ai');
        }

        try {
            $plan = GetPlan::run($args['slug'] ?? '', (int) $workspaceId);
        } catch (\InvalidArgumentException $e) {
            return $this->error($e->getMessage());
        }

        $format = $args['format'] ?? 'json';

        if ($format === 'markdown') {
            return $this->success(['markdown' => $plan->toMarkdown()]);
        }

        return $this->success(['plan' => $plan->toMcpContext()]);
    }
}
