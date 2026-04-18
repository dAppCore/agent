<?php

declare(strict_types=1);

namespace Core\Mod\Agentic\Mcp\Tools\Agent\Phase;

use Core\Mod\Agentic\Actions\Phase\AddCheckpoint;
use Core\Mod\Agentic\Mcp\Tools\Agent\AgentTool;

/**
 * Add a checkpoint note to a phase.
 */
class PhaseAddCheckpoint extends AgentTool
{
    protected string $category = 'phase';

    protected array $scopes = ['write'];

    public function name(): string
    {
        return 'phase_add_checkpoint';
    }

    public function description(): string
    {
        return 'Add a checkpoint note to a phase';
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
                'note' => [
                    'type' => 'string',
                    'description' => 'Checkpoint note',
                ],
                'context' => [
                    'type' => 'object',
                    'description' => 'Additional context data',
                ],
            ],
            'required' => ['plan_slug', 'phase', 'note'],
        ];
    }

    public function handle(array $args, array $context = []): array
    {
        $workspaceId = $context['workspace_id'] ?? null;
        if ($workspaceId === null) {
            return $this->error('workspace_id is required');
        }

        try {
            $phase = AddCheckpoint::run(
                $args['plan_slug'] ?? '',
                $args['phase'] ?? '',
                $args['note'] ?? '',
                (int) $workspaceId,
                $args['context'] ?? [],
            );

            return $this->success([
                'checkpoints' => $phase->getCheckpoints(),
            ]);
        } catch (\InvalidArgumentException $e) {
            return $this->error($e->getMessage());
        }
    }
}
