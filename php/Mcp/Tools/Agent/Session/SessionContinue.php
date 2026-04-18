<?php

declare(strict_types=1);

namespace Core\Mod\Agentic\Mcp\Tools\Agent\Session;

use Core\Mod\Agentic\Actions\Session\ContinueSession;
use Core\Mod\Agentic\Mcp\Tools\Agent\AgentTool;

/**
 * Continue from a previous session (multi-agent handoff).
 */
class SessionContinue extends AgentTool
{
    protected string $category = 'session';

    protected array $scopes = ['write'];

    public function name(): string
    {
        return 'session_continue';
    }

    public function description(): string
    {
        return 'Continue from a previous session (multi-agent handoff)';
    }

    public function inputSchema(): array
    {
        return [
            'type' => 'object',
            'properties' => [
                'previous_session_id' => [
                    'type' => 'string',
                    'description' => 'Session ID to continue from',
                ],
                'agent_type' => [
                    'type' => 'string',
                    'description' => 'New agent type taking over',
                ],
            ],
            'required' => ['previous_session_id', 'agent_type'],
        ];
    }

    public function handle(array $args, array $context = []): array
    {
        try {
            $session = ContinueSession::run(
                $args['previous_session_id'] ?? '',
                $args['agent_type'] ?? '',
            );

            $inheritedContext = $session->context_summary ?? [];

            return $this->success([
                'session' => [
                    'session_id' => $session->session_id,
                    'agent_type' => $session->agent_type,
                    'status' => $session->status,
                    'plan' => $session->plan?->slug,
                ],
                'continued_from' => $inheritedContext['continued_from'] ?? null,
                'previous_agent' => $inheritedContext['previous_agent'] ?? null,
                'handoff_notes' => $inheritedContext['handoff_notes'] ?? null,
                'inherited_context' => $inheritedContext['inherited_context'] ?? null,
            ]);
        } catch (\InvalidArgumentException $e) {
            return $this->error($e->getMessage());
        }
    }
}
