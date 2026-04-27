<?php

declare(strict_types=1);

namespace Core\Mod\Agentic\Mcp\Tools\Agent\Brain;

use Core\Mcp\Dependencies\ToolDependency;
use Core\Mod\Agentic\Actions\Brain\RememberKnowledge;
use Core\Mod\Agentic\Mcp\Tools\Agent\AgentTool;
use Core\Mod\Agentic\Models\BrainMemory;

/**
 * Store a memory in the shared OpenBrain knowledge store.
 *
 * Agents use this tool to persist decisions, observations, conventions,
 * and other knowledge so that other agents can recall it later.
 */
class BrainRemember extends AgentTool
{
    protected string $category = 'brain';

    protected array $scopes = ['write'];

    public function dependencies(): array
    {
        return [
            ToolDependency::contextExists('workspace_id', 'Workspace context required to store memories'),
        ];
    }

    public function name(): string
    {
        return 'brain_remember';
    }

    public function description(): string
    {
        return 'Store a memory in the shared OpenBrain knowledge store. Use this to persist decisions, observations, conventions, research, plans, bugs, or architecture knowledge for other agents.';
    }

    public function inputSchema(): array
    {
        return [
            'type' => 'object',
            'properties' => [
                'content' => [
                    'type' => 'string',
                    'description' => 'The knowledge to remember (max 50,000 characters)',
                    'maxLength' => 50000,
                ],
                'type' => [
                    'type' => 'string',
                    'description' => 'Memory type classification',
                    'enum' => BrainMemory::VALID_TYPES,
                ],
                'tags' => [
                    'type' => 'array',
                    'items' => ['type' => 'string'],
                    'description' => 'Optional tags for categorisation',
                ],
                'org' => [
                    'type' => 'string',
                    'description' => 'Optional organisation scope',
                    'maxLength' => 128,
                ],
                'project' => [
                    'type' => 'string',
                    'description' => 'Optional project scope (e.g. repo name)',
                ],
                'confidence' => [
                    'type' => 'number',
                    'description' => 'Confidence level from 0.0 to 1.0 (default: 0.8)',
                    'minimum' => 0.0,
                    'maximum' => 1.0,
                ],
                'supersedes' => [
                    'type' => 'string',
                    'format' => 'uuid',
                    'description' => 'UUID of an older memory this one replaces',
                ],
                'expires_in' => [
                    'type' => 'integer',
                    'description' => 'Hours until this memory expires (null = never)',
                    'minimum' => 1,
                ],
            ],
            'required' => ['content', 'type'],
        ];
    }

    public function handle(array $args, array $context = []): array
    {
        $workspaceId = $context['workspace_id'] ?? null;
        if ($workspaceId === null) {
            return $this->error('workspace_id is required. Ensure you have authenticated with a valid API key. See: https://host.uk.com/ai');
        }

        $agentId = $context['agent_id'] ?? $context['session_id'] ?? 'anonymous';
        $payload = $args;
        $payload['org'] = $this->optionalString($args, 'org', null, 128);

        return $this->withCircuitBreaker('brain', function () use ($payload, $workspaceId, $agentId) {
            $memory = RememberKnowledge::run($payload, (int) $workspaceId, $agentId);

            return $this->success([
                'memory' => $memory->toMcpContext(),
            ]);
        }, fn () => $this->error('Brain service temporarily unavailable. Memory could not be stored.', 'service_unavailable'));
    }
}
