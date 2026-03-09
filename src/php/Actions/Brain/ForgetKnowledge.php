<?php

declare(strict_types=1);

namespace Core\Mod\Agentic\Actions\Brain;

use Core\Actions\Action;
use Core\Mod\Agentic\Models\BrainMemory;
use Core\Mod\Agentic\Services\BrainService;
use Illuminate\Support\Facades\Log;

/**
 * Remove a memory from the shared OpenBrain knowledge store.
 *
 * Deletes from both MariaDB and Qdrant. Workspace-scoped.
 *
 * Usage:
 *   ForgetKnowledge::run('uuid-here', 1, 'virgil', 'outdated info');
 */
class ForgetKnowledge
{
    use Action;

    public function __construct(
        private BrainService $brain,
    ) {}

    /**
     * @return array{forgotten: string, type: string}
     *
     * @throws \InvalidArgumentException
     * @throws \RuntimeException
     */
    public function handle(string $id, int $workspaceId, string $agentId = 'anonymous', ?string $reason = null): array
    {
        if ($id === '') {
            throw new \InvalidArgumentException('id is required');
        }

        $memory = BrainMemory::where('id', $id)
            ->where('workspace_id', $workspaceId)
            ->first();

        if (! $memory) {
            throw new \InvalidArgumentException("Memory '{$id}' not found in this workspace");
        }

        Log::info('OpenBrain: memory forgotten', [
            'id' => $id,
            'type' => $memory->type,
            'agent_id' => $agentId,
            'reason' => $reason,
        ]);

        $this->brain->forget($id);

        return [
            'forgotten' => $id,
            'type' => $memory->type,
        ];
    }
}
