<?php

declare(strict_types=1);

namespace Core\Mod\Agentic\Mcp\Tools\Agent\Brain;

use Core\Mcp\Dependencies\ToolDependency;
use Core\Mod\Agentic\Actions\Brain\ForgetKnowledge;
use Core\Mod\Agentic\Mcp\Tools\Agent\AgentTool;
use Core\Mod\Agentic\Models\BrainMemory;

/**
 * Remove a memory from the shared OpenBrain knowledge store.
 *
 * Deletes the memory from both MariaDB and Qdrant.
 * Workspace-scoped: agents can only forget memories in their own workspace.
 */
class BrainForget extends AgentTool
{
    protected string $category = 'brain';

    protected array $scopes = ['write'];

    public function dependencies(): array
    {
        return [
            ToolDependency::contextExists('workspace_id', 'Workspace context required to forget memories'),
        ];
    }

    public function name(): string
    {
        return 'brain_forget';
    }

    public function description(): string
    {
        return 'Remove a memory from the shared OpenBrain knowledge store. Permanently deletes from both database and vector index.';
    }

    public function inputSchema(): array
    {
        return [
            'type' => 'object',
            'properties' => [
                'id' => [
                    'type' => 'string',
                    'format' => 'uuid',
                    'description' => 'UUID of the memory to remove',
                ],
                'reason' => [
                    'type' => 'string',
                    'description' => 'Optional reason for forgetting this memory',
                    'maxLength' => 500,
                ],
            ],
            'required' => ['id'],
        ];
    }

    public function handle(array $args, array $context = []): array
    {
        $workspaceId = $context['workspace_id'] ?? null;
        if ($workspaceId === null) {
            return $this->error('workspace_id is required. Ensure you have authenticated with a valid API key. See: https://host.uk.com/ai');
        }

        $id = $args['id'] ?? '';
        $reason = $this->optionalString($args, 'reason', null, 500);
        $agentId = $context['agent_id'] ?? $context['session_id'] ?? 'anonymous';

        return $this->withCircuitBreaker('brain', function () use ($id, $workspaceId, $agentId, $reason) {
            $result = ForgetKnowledge::run($id, (int) $workspaceId, $agentId, $reason);

            return $this->success($result);
        }, fn () => $this->error('Brain service temporarily unavailable. Memory could not be removed.', 'service_unavailable'));
    }
}
