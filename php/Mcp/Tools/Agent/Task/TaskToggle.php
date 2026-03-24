<?php

declare(strict_types=1);

namespace Core\Mod\Agentic\Mcp\Tools\Agent\Task;

use Core\Mcp\Dependencies\ToolDependency;
use Core\Mod\Agentic\Actions\Task\ToggleTask;
use Core\Mod\Agentic\Mcp\Tools\Agent\AgentTool;

/**
 * Toggle a task completion status.
 */
class TaskToggle extends AgentTool
{
    protected string $category = 'task';

    protected array $scopes = ['write'];

    /**
     * Get the dependencies for this tool.
     *
     * @return array<ToolDependency>
     */
    public function dependencies(): array
    {
        return [
            ToolDependency::entityExists('plan', 'Plan must exist', ['arg_key' => 'plan_slug']),
        ];
    }

    public function name(): string
    {
        return 'task_toggle';
    }

    public function description(): string
    {
        return 'Toggle a task completion status';
    }

    public function inputSchema(): array
    {
        return [
            'type' => 'object',
            'properties' => [
                'plan_slug' => [
                    'type' => 'string',
                    'description' => 'Plan slug identifier',
                ],
                'phase' => [
                    'type' => 'string',
                    'description' => 'Phase identifier (number or name)',
                ],
                'task_index' => [
                    'type' => 'integer',
                    'description' => 'Task index (0-based)',
                ],
            ],
            'required' => ['plan_slug', 'phase', 'task_index'],
        ];
    }

    public function handle(array $args, array $context = []): array
    {
        $workspaceId = $context['workspace_id'] ?? null;
        if ($workspaceId === null) {
            return $this->error('workspace_id is required');
        }

        try {
            $result = ToggleTask::run(
                $args['plan_slug'] ?? '',
                $args['phase'] ?? '',
                (int) ($args['task_index'] ?? 0),
                (int) $workspaceId,
            );

            return $this->success($result);
        } catch (\InvalidArgumentException $e) {
            return $this->error($e->getMessage());
        }
    }
}
