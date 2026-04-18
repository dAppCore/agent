<?php

declare(strict_types=1);

namespace Core\Mod\Agentic\Mcp\Tools\Agent\Session;

use Core\Mod\Agentic\Mcp\Tools\Agent\AgentTool;
use Core\Mod\Agentic\Models\AgentSession;

/**
 * Prepare session for handoff to another agent.
 */
class SessionHandoff extends AgentTool
{
    protected string $category = 'session';

    protected array $scopes = ['write'];

    public function name(): string
    {
        return 'session_handoff';
    }

    public function description(): string
    {
        return 'Prepare session for handoff to another agent';
    }

    public function inputSchema(): array
    {
        return [
            'type' => 'object',
            'properties' => [
                'summary' => [
                    'type' => 'string',
                    'description' => 'Summary of work done',
                ],
                'next_steps' => [
                    'type' => 'array',
                    'description' => 'Recommended next steps',
                    'items' => ['type' => 'string'],
                ],
                'blockers' => [
                    'type' => 'array',
                    'description' => 'Any blockers encountered',
                    'items' => ['type' => 'string'],
                ],
                'context_for_next' => [
                    'type' => 'object',
                    'description' => 'Context to pass to next agent',
                ],
            ],
            'required' => ['summary'],
        ];
    }

    public function handle(array $args, array $context = []): array
    {
        try {
            $summary = $this->require($args, 'summary');
        } catch (\InvalidArgumentException $e) {
            return $this->error($e->getMessage());
        }

        $sessionId = $context['session_id'] ?? null;

        if (! $sessionId) {
            return $this->error('No active session. Call session_start first.');
        }

        $session = AgentSession::where('session_id', $sessionId)->first();

        if (! $session) {
            return $this->error('Session not found');
        }

        $session->prepareHandoff(
            $summary,
            $this->optional($args, 'next_steps', []),
            $this->optional($args, 'blockers', []),
            $this->optional($args, 'context_for_next', [])
        );

        return $this->success([
            'handoff_context' => $session->getHandoffContext(),
        ]);
    }
}
