<?php

declare(strict_types=1);

namespace Core\Mod\Agentic\Mcp\Tools\Agent\Session;

use Core\Mcp\Dependencies\ToolDependency;
use Core\Mod\Agentic\Actions\Session\StartSession;
use Core\Mod\Agentic\Mcp\Tools\Agent\AgentTool;

/**
 * Start a new agent session for a plan.
 */
class SessionStart extends AgentTool
{
    protected string $category = 'session';

    protected array $scopes = ['write'];

    /**
     * Get the dependencies for this tool.
     *
     * Workspace context is needed unless a plan_slug is provided
     * (in which case workspace is inferred from the plan).
     *
     * @return array<ToolDependency>
     */
    public function dependencies(): array
    {
        // Soft dependency - workspace can come from plan
        return [
            ToolDependency::contextExists('workspace_id', 'Workspace context required (or provide plan_slug)')
                ->asOptional(),
        ];
    }

    public function name(): string
    {
        return 'session_start';
    }

    public function description(): string
    {
        return 'Start a new agent session for a plan';
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
                'agent_type' => [
                    'type' => 'string',
                    'description' => 'Type of agent (e.g., opus, sonnet, haiku)',
                ],
                'context' => [
                    'type' => 'object',
                    'description' => 'Initial session context',
                ],
            ],
            'required' => ['agent_type'],
        ];
    }

    public function handle(array $args, array $context = []): array
    {
        $workspaceId = $context['workspace_id'] ?? null;
        if ($workspaceId === null) {
            return $this->error('workspace_id is required. Ensure you have authenticated with a valid API key and started a session, or provide a valid plan_slug to infer workspace context. See: https://host.uk.com/ai');
        }

        try {
            $session = StartSession::run(
                $args['agent_type'] ?? '',
                $args['plan_slug'] ?? null,
                (int) $workspaceId,
                $args['context'] ?? [],
            );

            return $this->success([
                'session' => [
                    'session_id' => $session->session_id,
                    'agent_type' => $session->agent_type,
                    'plan' => $session->plan?->slug,
                    'status' => $session->status,
                ],
            ]);
        } catch (\InvalidArgumentException $e) {
            return $this->error($e->getMessage());
        }
    }
}
