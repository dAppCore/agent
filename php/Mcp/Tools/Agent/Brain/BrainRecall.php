<?php

declare(strict_types=1);

namespace Core\Mod\Agentic\Mcp\Tools\Agent\Brain;

use Core\Mcp\Dependencies\ToolDependency;
use Core\Mod\Agentic\Actions\Brain\RecallKnowledge;
use Core\Mod\Agentic\Mcp\Tools\Agent\AgentTool;
use Core\Mod\Agentic\Models\BrainMemory;

/**
 * Semantic search across the shared OpenBrain knowledge store.
 *
 * Uses vector similarity to find memories relevant to a natural
 * language query, with optional filtering by project, type, agent,
 * or minimum confidence.
 */
class BrainRecall extends AgentTool
{
    protected string $category = 'brain';

    protected array $scopes = ['read'];

    public function dependencies(): array
    {
        return [
            ToolDependency::contextExists('workspace_id', 'Workspace context required to recall memories'),
        ];
    }

    public function name(): string
    {
        return 'brain_recall';
    }

    public function description(): string
    {
        return 'Semantic search across the shared OpenBrain knowledge store. Returns memories ranked by similarity to your query, with optional filtering.';
    }

    public function inputSchema(): array
    {
        return [
            'type' => 'object',
            'properties' => [
                'query' => [
                    'type' => 'string',
                    'description' => 'Natural language search query (max 2,000 characters)',
                    'maxLength' => 2000,
                ],
                'top_k' => [
                    'type' => 'integer',
                    'description' => 'Number of results to return (default: 5, max: 20)',
                    'minimum' => 1,
                    'maximum' => 20,
                    'default' => 5,
                ],
                'filter' => [
                    'type' => 'object',
                    'description' => 'Optional filters to narrow results',
                    'properties' => [
                        'org' => [
                            'type' => 'string',
                            'description' => 'Filter by organisation scope',
                        ],
                        'project' => [
                            'type' => 'string',
                            'description' => 'Filter by project scope',
                        ],
                        'type' => [
                            'oneOf' => [
                                ['type' => 'string', 'enum' => BrainMemory::VALID_TYPES],
                                [
                                    'type' => 'array',
                                    'items' => ['type' => 'string', 'enum' => BrainMemory::VALID_TYPES],
                                ],
                            ],
                            'description' => 'Filter by memory type (single or array)',
                        ],
                        'agent_id' => [
                            'type' => 'string',
                            'description' => 'Filter by originating agent',
                        ],
                        'min_confidence' => [
                            'type' => 'number',
                            'description' => 'Minimum confidence threshold (0.0-1.0)',
                            'minimum' => 0.0,
                            'maximum' => 1.0,
                        ],
                    ],
                ],
            ],
            'required' => ['query'],
        ];
    }

    public function handle(array $args, array $context = []): array
    {
        $workspaceId = $context['workspace_id'] ?? null;
        if ($workspaceId === null) {
            return $this->error('workspace_id is required. Ensure you have authenticated with a valid API key. See: https://host.uk.com/ai');
        }

        $query = $args['query'] ?? '';
        $topK = $this->optionalInt($args, 'top_k', 5, 1, 20);
        $filter = $this->optional($args, 'filter', []);

        if (! is_array($filter)) {
            return $this->error('filter must be an object');
        }

        return $this->withCircuitBreaker('brain', function () use ($query, $workspaceId, $filter, $topK) {
            $result = RecallKnowledge::run($query, (int) $workspaceId, $filter, $topK);

            return $this->success([
                'count' => $result['count'],
                'memories' => $result['memories'],
                'scores' => $result['scores'],
            ]);
        }, fn () => $this->error('Brain service temporarily unavailable. Recall failed.', 'service_unavailable'));
    }
}
