<?php

declare(strict_types=1);

namespace Core\Mod\Agentic\Mcp\Tools\Agent\Brain;

use Core\Mcp\Dependencies\ToolDependency;
use Core\Mod\Agentic\Mcp\Tools\Agent\AgentTool;
use Core\Mod\Agentic\Models\BrainMemory;

/**
 * List memories in the shared OpenBrain knowledge store.
 *
 * Pure MariaDB query using model scopes -- no vector search.
 * Useful for browsing what an agent or project has stored.
 */
class BrainList extends AgentTool
{
    protected string $category = 'brain';

    protected array $scopes = ['read'];

    public function dependencies(): array
    {
        return [
            ToolDependency::contextExists('workspace_id', 'Workspace context required to list memories'),
        ];
    }

    public function name(): string
    {
        return 'brain_list';
    }

    public function description(): string
    {
        return 'List memories in the shared OpenBrain knowledge store. Supports filtering by organisation, project, type, and agent. No vector search -- use brain_recall for semantic queries.';
    }

    public function inputSchema(): array
    {
        return [
            'type' => 'object',
            'properties' => [
                'org' => [
                    'type' => 'string',
                    'description' => 'Filter by organisation scope',
                    'maxLength' => 128,
                ],
                'project' => [
                    'type' => 'string',
                    'description' => 'Filter by project scope',
                ],
                'type' => [
                    'type' => 'string',
                    'description' => 'Filter by memory type',
                    'enum' => BrainMemory::VALID_TYPES,
                ],
                'agent_id' => [
                    'type' => 'string',
                    'description' => 'Filter by originating agent',
                ],
                'limit' => [
                    'type' => 'integer',
                    'description' => 'Maximum results to return (default: 20, max: 100)',
                    'minimum' => 1,
                    'maximum' => 100,
                    'default' => 20,
                ],
            ],
        ];
    }

    public function handle(array $args, array $context = []): array
    {
        $workspaceId = $context['workspace_id'] ?? null;
        if ($workspaceId === null) {
            return $this->error('workspace_id is required. Ensure you have authenticated with a valid API key. See: https://host.uk.com/ai');
        }

        $org = $this->optionalString($args, 'org', null, 128);
        $project = $this->optionalString($args, 'project', null);
        $agentId = $this->optionalString($args, 'agent_id', null);
        $type = $this->optionalEnum($args, 'type', BrainMemory::VALID_TYPES);
        $limit = $this->optionalInt($args, 'limit', 20, 1, 100);

        $query = BrainMemory::forWorkspace((int) $workspaceId)
            ->active()
            ->latestVersions()
            ->forOrg($org)
            ->forProject($project)
            ->byAgent($agentId);

        if ($type !== null) {
            $query->ofType($type);
        }

        $memories = $query->orderByDesc('created_at')
            ->limit($limit)
            ->get();

        return $this->success([
            'memories' => $memories->map(fn (BrainMemory $memory): array => $memory->toMcpContext())->all(),
            'count' => $memories->count(),
        ]);
    }
}
