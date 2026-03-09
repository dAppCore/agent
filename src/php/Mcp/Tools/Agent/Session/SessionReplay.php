<?php

declare(strict_types=1);

namespace Core\Mod\Agentic\Mcp\Tools\Agent\Session;

use Core\Mod\Agentic\Mcp\Tools\Agent\AgentTool;
use Core\Mod\Agentic\Services\AgentSessionService;

/**
 * Replay a session by creating a new session with the original's context.
 *
 * This tool reconstructs the state from a session's work log and creates
 * a new active session, allowing an agent to continue from where the
 * original session left off.
 */
class SessionReplay extends AgentTool
{
    protected string $category = 'session';

    protected array $scopes = ['write'];

    public function name(): string
    {
        return 'session_replay';
    }

    public function description(): string
    {
        return 'Replay a session - creates a new session with the original\'s reconstructed context from its work log';
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
                'agent_type' => [
                    'type' => 'string',
                    'description' => 'Agent type for the new session (defaults to original session\'s agent type)',
                ],
                'context_only' => [
                    'type' => 'boolean',
                    'description' => 'If true, only return the replay context without creating a new session',
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

        $agentType = $this->optional($args, 'agent_type');
        $contextOnly = $this->optional($args, 'context_only', false);

        return $this->withCircuitBreaker('agentic', function () use ($sessionId, $agentType, $contextOnly) {
            $sessionService = app(AgentSessionService::class);

            // If only context requested, return the replay context
            if ($contextOnly) {
                $replayContext = $sessionService->getReplayContext($sessionId);

                if (! $replayContext) {
                    return $this->error("Session not found: {$sessionId}");
                }

                return $this->success([
                    'replay_context' => $replayContext,
                ]);
            }

            // Create a new replay session
            $newSession = $sessionService->replay($sessionId, $agentType);

            if (! $newSession) {
                return $this->error("Session not found: {$sessionId}");
            }

            return $this->success([
                'session' => [
                    'session_id' => $newSession->session_id,
                    'agent_type' => $newSession->agent_type,
                    'status' => $newSession->status,
                    'plan' => $newSession->plan?->slug,
                ],
                'replayed_from' => $sessionId,
                'context_summary' => $newSession->context_summary,
            ]);
        }, fn () => $this->error('Agentic service temporarily unavailable.', 'service_unavailable'));
    }
}
