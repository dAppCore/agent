<?php

declare(strict_types=1);

namespace Core\Mod\Agentic\Mcp\Tools\Agent\Session;

use Core\Mod\Agentic\Actions\Session\EndSession;
use Core\Mod\Agentic\Mcp\Tools\Agent\AgentTool;

/**
 * End the current session.
 */
class SessionEnd extends AgentTool
{
    protected string $category = 'session';

    protected array $scopes = ['write'];

    public function name(): string
    {
        return 'session_end';
    }

    public function description(): string
    {
        return 'End the current session';
    }

    public function inputSchema(): array
    {
        return [
            'type' => 'object',
            'properties' => [
                'status' => [
                    'type' => 'string',
                    'description' => 'Final session status',
                    'enum' => ['completed', 'handed_off', 'paused', 'failed'],
                ],
                'summary' => [
                    'type' => 'string',
                    'description' => 'Final summary',
                ],
                'handoff_notes' => [
                    'type' => 'object',
                    'description' => 'Optional handoff details for the next agent',
                ],
            ],
            'required' => ['status'],
        ];
    }

    public function handle(array $args, array $context = []): array
    {
        $sessionId = $context['session_id'] ?? null;
        if (! $sessionId) {
            return $this->error('No active session');
        }

        try {
            $session = EndSession::run(
                $sessionId,
                $args['status'] ?? '',
                $args['summary'] ?? null,
                $args['handoff_notes'] ?? null,
            );

            return $this->success([
                'session' => [
                    'session_id' => $session->session_id,
                    'status' => $session->status,
                    'duration' => $session->getDurationFormatted(),
                ],
            ]);
        } catch (\InvalidArgumentException $e) {
            return $this->error($e->getMessage());
        }
    }
}
