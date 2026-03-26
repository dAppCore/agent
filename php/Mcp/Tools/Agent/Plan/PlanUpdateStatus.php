<?php

declare(strict_types=1);

namespace Core\Mod\Agentic\Mcp\Tools\Agent\Plan;

use Core\Mod\Agentic\Actions\Plan\UpdatePlanStatus;
use Core\Mod\Agentic\Mcp\Tools\Agent\AgentTool;

/**
 * Update the status of a plan.
 */
class PlanUpdateStatus extends AgentTool
{
    protected string $category = 'plan';

    protected array $scopes = ['write'];

    public function name(): string
    {
        return 'plan_update_status';
    }

    public function description(): string
    {
        return 'Update the status of a plan';
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
                'status' => [
                    'type' => 'string',
                    'description' => 'New status',
                    'enum' => ['draft', 'active', 'paused', 'completed'],
                ],
            ],
            'required' => ['slug', 'status'],
        ];
    }

    public function handle(array $args, array $context = []): array
    {
        $workspaceId = $context['workspace_id'] ?? null;
        if ($workspaceId === null) {
            return $this->error('workspace_id is required');
        }

        try {
            $plan = UpdatePlanStatus::run(
                $args['slug'] ?? '',
                $args['status'] ?? '',
                (int) $workspaceId,
            );

            return $this->success([
                'plan' => [
                    'slug' => $plan->slug,
                    'status' => $plan->status,
                ],
            ]);
        } catch (\InvalidArgumentException $e) {
            return $this->error($e->getMessage());
        }
    }
}
