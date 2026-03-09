<?php

declare(strict_types=1);

namespace Core\Mod\Agentic\Actions\Brain;

use Core\Actions\Action;
use Core\Mod\Agentic\Models\BrainMemory;
use Core\Mod\Agentic\Services\BrainService;

/**
 * Semantic search across the shared OpenBrain knowledge store.
 *
 * Uses vector similarity to find memories relevant to a natural
 * language query, with optional filtering by project, type, agent,
 * or minimum confidence.
 *
 * Usage:
 *   $results = RecallKnowledge::run('how does auth work?', 1);
 */
class RecallKnowledge
{
    use Action;

    public function __construct(
        private BrainService $brain,
    ) {}

    /**
     * @param  array{project?: string, type?: string|array, agent_id?: string, min_confidence?: float}  $filter
     * @return array{memories: array, scores: array<string, float>, count: int}
     *
     * @throws \InvalidArgumentException
     * @throws \RuntimeException
     */
    public function handle(string $query, int $workspaceId, array $filter = [], int $topK = 5): array
    {
        if ($query === '') {
            throw new \InvalidArgumentException('query is required and must be a non-empty string');
        }
        if (mb_strlen($query) > 2000) {
            throw new \InvalidArgumentException('query must not exceed 2,000 characters');
        }

        if ($topK < 1 || $topK > 20) {
            throw new \InvalidArgumentException('top_k must be between 1 and 20');
        }

        if (isset($filter['type'])) {
            $typeValue = $filter['type'];
            $validTypes = BrainMemory::VALID_TYPES;

            if (is_string($typeValue)) {
                if (! in_array($typeValue, $validTypes, true)) {
                    throw new \InvalidArgumentException(
                        sprintf('filter.type must be one of: %s', implode(', ', $validTypes))
                    );
                }
            } elseif (is_array($typeValue)) {
                foreach ($typeValue as $t) {
                    if (! is_string($t) || ! in_array($t, $validTypes, true)) {
                        throw new \InvalidArgumentException(
                            sprintf('Each filter.type value must be one of: %s', implode(', ', $validTypes))
                        );
                    }
                }
            }
        }

        if (isset($filter['min_confidence'])) {
            $mc = $filter['min_confidence'];
            if (! is_numeric($mc) || $mc < 0.0 || $mc > 1.0) {
                throw new \InvalidArgumentException('filter.min_confidence must be between 0.0 and 1.0');
            }
        }

        $result = $this->brain->recall($query, $topK, $filter, $workspaceId);

        return [
            'memories' => $result['memories'],
            'scores' => $result['scores'],
            'count' => count($result['memories']),
        ];
    }
}
