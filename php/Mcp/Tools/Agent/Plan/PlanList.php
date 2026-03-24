<?php

declare(strict_types=1);

namespace Core\Mod\Agentic\Mcp\Tools\Agent\Plan;

use Core\Mcp\Dependencies\ToolDependency;
use Core\Mod\Agentic\Actions\Plan\ListPlans;
use Core\Mod\Agentic\Mcp\Tools\Agent\AgentTool;

/**
 * List all work plans with their current status and progress.
 */
class PlanList extends AgentTool
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
        return 'plan_list';
    }

    public function description(): string
    {
        return 'List all work plans with their current status and progress';
    }

    public function inputSchema(): array
    {
        return [
            'type' => 'object',
            'properties' => [
                'status' => [
                    'type' => 'string',
                    'description' => 'Filter by status (draft, active, paused, completed, archived)',
                    'enum' => ['draft', 'active', 'paused', 'completed', 'archived'],
                ],
                'include_archived' => [
                    'type' => 'boolean',
                    'description' => 'Include archived plans (default: false)',
                ],
            ],
        ];
    }

    public function handle(array $args, array $context = []): array
    {
        $workspaceId = $context['workspace_id'] ?? null;
        if ($workspaceId === null) {
            return $this->error('workspace_id is required. Ensure you have authenticated with a valid API key and started a session. See: https://host.uk.com/ai');
        }

        try {
            $plans = ListPlans::run(
                (int) $workspaceId,
                $args['status'] ?? null,
                (bool) ($args['include_archived'] ?? false),
            );

            return $this->success([
                'plans' => $plans->map(fn ($plan) => [
                    'slug' => $plan->slug,
                    'title' => $plan->title,
                    'status' => $plan->status,
                    'progress' => $plan->getProgress(),
                    'updated_at' => $plan->updated_at->toIso8601String(),
                ])->all(),
                'total' => $plans->count(),
            ]);
        } catch (\InvalidArgumentException $e) {
            return $this->error($e->getMessage());
        }
    }
}
