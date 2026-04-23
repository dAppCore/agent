<?php

declare(strict_types=1);

namespace Core\Mod\Agentic\Mcp\Tools\Agent\Session;

use Core\Mod\Agentic\Mcp\Tools\Agent\AgentTool;
use Core\Mod\Agentic\Models\AgentSession;

/**
 * Record an artifact created/modified during the session.
 */
class SessionArtifact extends AgentTool
{
    protected string $category = 'session';

    protected array $scopes = ['write'];

    public function name(): string
    {
        return 'session_artifact';
    }

    public function description(): string
    {
        return 'Record an artifact created/modified during the session';
    }

    public function inputSchema(): array
    {
        return [
            'type' => 'object',
            'properties' => [
                'path' => [
                    'type' => 'string',
                    'description' => 'File or resource path',
                ],
                'action' => [
                    'type' => 'string',
                    'description' => 'Action performed',
                    'enum' => ['created', 'modified', 'deleted', 'reviewed'],
                ],
                'description' => [
                    'type' => 'string',
                    'description' => 'Description of changes',
                ],
            ],
            'required' => ['path', 'action'],
        ];
    }

    public function handle(array $args, array $context = []): array
    {
        try {
            $path = $this->require($args, 'path');
            $action = $this->require($args, 'action');
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

        $description = $this->optional($args, 'description');
        $metadata = $description !== null ? ['description' => $description] : null;

        $session->addArtifact($path, $action, $metadata);

        return $this->success(['artifact' => $path]);
    }
}
