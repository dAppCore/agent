<?php

declare(strict_types=1);

namespace Core\Mod\Agentic\Mcp\Tools\Agent\Task;

use Core\Mcp\Dependencies\ToolDependency;
use Core\Mod\Agentic\Actions\Task\UpdateTask;
use Core\Mod\Agentic\Mcp\Tools\Agent\AgentTool;

/**
 * Update task details (status, notes).
 */
class TaskUpdate extends AgentTool
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
        return 'task_update';
    }

    public function description(): string
    {
        return 'Update task details (status, notes)';
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
                'status' => [
                    'type' => 'string',
                    'description' => 'New status',
                    'enum' => ['pending', 'in_progress', 'completed', 'blocked', 'skipped'],
                ],
                'notes' => [
                    'type' => 'string',
                    'description' => 'Task notes',
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
            $result = UpdateTask::run(
                $args['plan_slug'] ?? '',
                $args['phase'] ?? '',
                (int) ($args['task_index'] ?? 0),
                (int) $workspaceId,
                $args['status'] ?? null,
                $args['notes'] ?? null,
            );

            return $this->success($result);
        } catch (\InvalidArgumentException $e) {
            return $this->error($e->getMessage());
        }
    }
}
