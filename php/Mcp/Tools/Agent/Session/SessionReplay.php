<?php

declare(strict_types=1);

namespace Core\Mod\Agentic\Mcp\Tools\Agent\Session;

use Core\Mod\Agentic\Mcp\Tools\Agent\AgentTool;
use Core\Mod\Agentic\Services\AgentSessionService;

/**
 * Get replay context for a stored session.
 *
 * This tool reconstructs the state from a session's work log so an agent
 * can resume analysis or hand off from a completed session.
 */
class SessionReplay extends AgentTool
{
    protected string $category = 'session';

    protected array $scopes = ['read'];

    public function name(): string
    {
        return 'session_replay';
    }

    public function description(): string
    {
        return 'Get replay context for a stored session';
    }

    public function inputSchema(): array
    {
        return [
            'type' => 'object',
            'properties' => [
                'session_id' => [
                    'type' => 'string',
                    'description' => 'Session ID to replay from',
                ],
            ],
            'required' => ['session_id'],
        ];
    }

    public function handle(array $args, array $context = []): array
    {
        try {
            $sessionId = $this->require($args, 'session_id');
        } catch (\InvalidArgumentException $e) {
            return $this->error($e->getMessage());
        }

        return $this->withCircuitBreaker('agentic', function () use ($sessionId) {
            $sessionService = app(AgentSessionService::class);
            $replayContext = $sessionService->getReplayContext($sessionId);

            if (! $replayContext) {
                return $this->error("Session not found: {$sessionId}");
            }

            return $this->success([
                'replay_context' => $replayContext,
            ]);
        }, fn () => $this->error('Agentic service temporarily unavailable.', 'service_unavailable'));
    }
}
