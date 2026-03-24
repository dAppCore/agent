<?php

declare(strict_types=1);

namespace Core\Mod\Agentic\Mcp\Tools\Agent\Session;

use Core\Mcp\Dependencies\ToolDependency;
use Core\Mod\Agentic\Mcp\Tools\Agent\AgentTool;
use Core\Mod\Agentic\Models\AgentSession;

/**
 * Log an entry in the current session.
 */
class SessionLog extends AgentTool
{
    protected string $category = 'session';

    protected array $scopes = ['write'];

    /**
     * Get the dependencies for this tool.
     *
     * @return array<ToolDependency>
     */
    public function dependencies(): array
    {
        return [
            ToolDependency::sessionState('session_id', 'Active session required. Call session_start first.'),
        ];
    }

    public function name(): string
    {
        return 'session_log';
    }

    public function description(): string
    {
        return 'Log an entry in the current session';
    }

    public function inputSchema(): array
    {
        return [
            'type' => 'object',
            'properties' => [
                'message' => [
                    'type' => 'string',
                    'description' => 'Log message',
                ],
                'type' => [
                    'type' => 'string',
                    'description' => 'Log type',
                    'enum' => ['info', 'progress', 'decision', 'error', 'checkpoint'],
                ],
                'data' => [
                    'type' => 'object',
                    'description' => 'Additional data to log',
                ],
            ],
            'required' => ['message'],
        ];
    }

    public function handle(array $args, array $context = []): array
    {
        try {
            $message = $this->require($args, 'message');
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

        $session->addWorkLogEntry(
            $message,
            $this->optional($args, 'type', 'info'),
            $this->optional($args, 'data', [])
        );

        return $this->success(['logged' => $message]);
    }
}
