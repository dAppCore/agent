<?php

declare(strict_types=1);

namespace Core\Mod\Agentic\Mcp\Tools\Agent\Phase;

use Core\Mcp\Dependencies\ToolDependency;
use Core\Mod\Agentic\Actions\Phase\UpdatePhaseStatus;
use Core\Mod\Agentic\Mcp\Tools\Agent\AgentTool;

/**
 * Update the status of a phase.
 */
class PhaseUpdateStatus extends AgentTool
{
    protected string $category = 'phase';

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
        return 'phase_update_status';
    }

    public function description(): string
    {
        return 'Update the status of a phase';
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
                'status' => [
                    'type' => 'string',
                    'description' => 'New status',
                    'enum' => ['pending', 'in_progress', 'completed', 'blocked', 'skipped'],
                ],
                'notes' => [
                    'type' => 'string',
                    'description' => 'Optional notes about the status change',
                ],
            ],
            'required' => ['plan_slug', 'phase', 'status'],
        ];
    }

    public function handle(array $args, array $context = []): array
    {
        $workspaceId = $context['workspace_id'] ?? null;
        if ($workspaceId === null) {
            return $this->error('workspace_id is required');
        }

        try {
            $phase = UpdatePhaseStatus::run(
                $args['plan_slug'] ?? '',
                $args['phase'] ?? '',
                $args['status'] ?? '',
                (int) $workspaceId,
                $args['notes'] ?? null,
            );

            return $this->success([
                'phase' => [
                    'order' => $phase->order,
                    'name' => $phase->name,
                    'status' => $phase->status,
                ],
            ]);
        } catch (\InvalidArgumentException $e) {
            return $this->error($e->getMessage());
        }
    }
}
