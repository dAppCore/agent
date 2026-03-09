<?php

declare(strict_types=1);

namespace Core\Mod\Agentic\Mcp\Tools\Agent\Plan;

use Core\Mcp\Dependencies\ToolDependency;
use Core\Mod\Agentic\Actions\Plan\CreatePlan;
use Core\Mod\Agentic\Mcp\Tools\Agent\AgentTool;

/**
 * Create a new work plan with phases and tasks.
 */
class PlanCreate extends AgentTool
{
    protected string $category = 'plan';

    protected array $scopes = ['write'];

    /**
     * Get the dependencies for this tool.
     *
     * @return array<ToolDependency>
     */
    public function dependencies(): array
    {
        return [
            ToolDependency::contextExists('workspace_id', 'Workspace context required'),
        ];
    }

    public function name(): string
    {
        return 'plan_create';
    }

    public function description(): string
    {
        return 'Create a new work plan with phases and tasks';
    }

    public function inputSchema(): array
    {
        return [
            'type' => 'object',
            'properties' => [
                'title' => [
                    'type' => 'string',
                    'description' => 'Plan title',
                ],
                'slug' => [
                    'type' => 'string',
                    'description' => 'URL-friendly identifier (auto-generated if not provided)',
                ],
                'description' => [
                    'type' => 'string',
                    'description' => 'Plan description',
                ],
                'context' => [
                    'type' => 'object',
                    'description' => 'Additional context (related files, dependencies, etc.)',
                ],
                'phases' => [
                    'type' => 'array',
                    'description' => 'Array of phase definitions with name, description, and tasks',
                    'items' => [
                        'type' => 'object',
                        'properties' => [
                            'name' => ['type' => 'string'],
                            'description' => ['type' => 'string'],
                            'tasks' => [
                                'type' => 'array',
                                'items' => ['type' => 'string'],
                            ],
                        ],
                    ],
                ],
            ],
            'required' => ['title'],
        ];
    }

    public function handle(array $args, array $context = []): array
    {
        $workspaceId = $context['workspace_id'] ?? null;
        if ($workspaceId === null) {
            return $this->error('workspace_id is required. Ensure you have authenticated with a valid API key and started a session. See: https://host.uk.com/ai');
        }

        try {
            $plan = CreatePlan::run($args, (int) $workspaceId);

            return $this->success([
                'plan' => [
                    'slug' => $plan->slug,
                    'title' => $plan->title,
                    'status' => $plan->status,
                    'phases' => $plan->agentPhases->count(),
                ],
            ]);
        } catch (\InvalidArgumentException $e) {
            return $this->error($e->getMessage());
        }
    }
}
