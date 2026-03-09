<?php

declare(strict_types=1);

namespace Core\Mod\Agentic\Mcp\Tools\Agent\Phase;

use Core\Mod\Agentic\Actions\Phase\GetPhase;
use Core\Mod\Agentic\Mcp\Tools\Agent\AgentTool;

/**
 * Get details of a specific phase within a plan.
 */
class PhaseGet extends AgentTool
{
    protected string $category = 'phase';

    protected array $scopes = ['read'];

    public function name(): string
    {
        return 'phase_get';
    }

    public function description(): string
    {
        return 'Get details of a specific phase within a plan';
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
            ],
            'required' => ['plan_slug', 'phase'],
        ];
    }

    public function handle(array $args, array $context = []): array
    {
        $workspaceId = $context['workspace_id'] ?? null;
        if ($workspaceId === null) {
            return $this->error('workspace_id is required');
        }

        try {
            $phase = GetPhase::run(
                $args['plan_slug'] ?? '',
                $args['phase'] ?? '',
                (int) $workspaceId,
            );

            return $this->success([
                'phase' => [
                    'order' => $phase->order,
                    'name' => $phase->name,
                    'description' => $phase->description,
                    'status' => $phase->status,
                    'tasks' => $phase->tasks,
                    'checkpoints' => $phase->getCheckpoints(),
                    'dependencies' => $phase->dependencies,
                ],
            ]);
        } catch (\InvalidArgumentException $e) {
            return $this->error($e->getMessage());
        }
    }
}
