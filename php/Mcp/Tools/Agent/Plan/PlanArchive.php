<?php

declare(strict_types=1);

namespace Core\Mod\Agentic\Mcp\Tools\Agent\Plan;

use Core\Mod\Agentic\Actions\Plan\ArchivePlan;
use Core\Mod\Agentic\Mcp\Tools\Agent\AgentTool;

/**
 * Archive a completed or abandoned plan.
 */
class PlanArchive extends AgentTool
{
    protected string $category = 'plan';

    protected array $scopes = ['write'];

    public function name(): string
    {
        return 'plan_archive';
    }

    public function description(): string
    {
        return 'Archive a completed or abandoned plan';
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
                'reason' => [
                    'type' => 'string',
                    'description' => 'Reason for archiving',
                ],
            ],
            'required' => ['slug'],
        ];
    }

    public function handle(array $args, array $context = []): array
    {
        $workspaceId = $context['workspace_id'] ?? null;
        if ($workspaceId === null) {
            return $this->error('workspace_id is required');
        }

        try {
            $plan = ArchivePlan::run(
                $args['slug'] ?? '',
                (int) $workspaceId,
                $args['reason'] ?? null,
            );

            return $this->success([
                'plan' => [
                    'slug' => $plan->slug,
                    'status' => 'archived',
                    'archived_at' => $plan->archived_at?->toIso8601String(),
                ],
            ]);
        } catch (\InvalidArgumentException $e) {
            return $this->error($e->getMessage());
        }
    }
}
