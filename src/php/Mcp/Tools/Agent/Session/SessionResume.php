<?php

declare(strict_types=1);

namespace Core\Mod\Agentic\Mcp\Tools\Agent\Session;

use Core\Mod\Agentic\Mcp\Tools\Agent\AgentTool;
use Core\Mod\Agentic\Services\AgentSessionService;

/**
 * Resume a paused or handed-off session.
 */
class SessionResume extends AgentTool
{
    protected string $category = 'session';

    protected array $scopes = ['write'];

    public function name(): string
    {
        return 'session_resume';
    }

    public function description(): string
    {
        return 'Resume a paused or handed-off session';
    }

    public function inputSchema(): array
    {
        return [
            'type' => 'object',
            'properties' => [
                'session_id' => [
                    'type' => 'string',
                    'description' => 'Session ID to resume',
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

        $sessionService = app(AgentSessionService::class);
        $session = $sessionService->resume($sessionId);

        if (! $session) {
            return $this->error("Session not found: {$sessionId}");
        }

        // Get handoff context if available
        $handoffContext = $session->getHandoffContext();

        return $this->success([
            'session' => [
                'session_id' => $session->session_id,
                'agent_type' => $session->agent_type,
                'status' => $session->status,
                'plan' => $session->plan?->slug,
                'duration' => $session->getDurationFormatted(),
            ],
            'handoff_context' => $handoffContext['handoff_notes'] ?? null,
            'recent_actions' => $handoffContext['recent_actions'] ?? [],
            'artifacts' => $handoffContext['artifacts'] ?? [],
        ]);
    }
}
