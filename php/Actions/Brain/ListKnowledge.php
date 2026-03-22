<?php

declare(strict_types=1);

namespace Core\Mod\Agentic\Actions\Brain;

use Core\Actions\Action;
use Core\Mod\Agentic\Models\BrainMemory;

/**
 * List memories in the shared OpenBrain knowledge store.
 *
 * Pure MariaDB query using model scopes — no vector search.
 * Use RecallKnowledge for semantic queries.
 *
 * Usage:
 *   $memories = ListKnowledge::run(1, ['type' => 'decision']);
 */
class ListKnowledge
{
    use Action;

    /**
     * @param  array{project?: string, type?: string, agent_id?: string, limit?: int}  $filter
     * @return array{memories: array, count: int}
     */
    public function handle(int $workspaceId, array $filter = []): array
    {
        $limit = min(max((int) ($filter['limit'] ?? 20), 1), 100);

        $query = BrainMemory::forWorkspace($workspaceId)
            ->active()
            ->latestVersions()
            ->forProject($filter['project'] ?? null)
            ->byAgent($filter['agent_id'] ?? null);

        $type = $filter['type'] ?? null;
        if ($type !== null) {
            if (is_string($type) && ! in_array($type, BrainMemory::VALID_TYPES, true)) {
                throw new \InvalidArgumentException(
                    sprintf('type must be one of: %s', implode(', ', BrainMemory::VALID_TYPES))
                );
            }
            $query->ofType($type);
        }

        $memories = $query->orderByDesc('created_at')
            ->limit($limit)
            ->get();

        return [
            'memories' => $memories->map(fn (BrainMemory $m) => $m->toMcpContext())->all(),
            'count' => $memories->count(),
        ];
    }
}
