<?php

declare(strict_types=1);

namespace Core\Mod\Agentic\Mcp\Tools\Agent\Session;

use Core\Mod\Agentic\Actions\Session\ListSessions;
use Core\Mod\Agentic\Mcp\Tools\Agent\AgentTool;

/**
 * List sessions, optionally filtered by status.
 */
class SessionList extends AgentTool
{
    protected string $category = 'session';

    protected array $scopes = ['read'];

    public function name(): string
    {
        return 'session_list';
    }

    public function description(): string
    {
        return 'List sessions, optionally filtered by status';
    }

    public function inputSchema(): array
    {
        return [
            'type' => 'object',
            'properties' => [
                'status' => [
                    'type' => 'string',
                    'description' => 'Filter by status',
                    'enum' => ['active', 'paused', 'completed', 'failed'],
                ],
                'plan_slug' => [
                    'type' => 'string',
                    'description' => 'Filter by plan slug',
                ],
                'limit' => [
                    'type' => 'integer',
                    'description' => 'Maximum number of sessions to return',
                ],
            ],
        ];
    }

    public function handle(array $args, array $context = []): array
    {
        $workspaceId = $context['workspace_id'] ?? null;
        if ($workspaceId === null) {
            return $this->error('workspace_id is required');
        }

        try {
            $sessions = ListSessions::run(
                (int) $workspaceId,
                $args['status'] ?? null,
                $args['plan_slug'] ?? null,
                isset($args['limit']) ? (int) $args['limit'] : null,
            );

            return $this->success([
                'sessions' => $sessions->map(fn ($session) => [
                    'session_id' => $session->session_id,
                    'agent_type' => $session->agent_type,
                    'status' => $session->status,
                    'plan' => $session->plan?->slug,
                    'duration' => $session->getDurationFormatted(),
                    'started_at' => $session->started_at->toIso8601String(),
                    'last_active_at' => $session->last_active_at->toIso8601String(),
                    'has_handoff' => ! empty($session->handoff_notes),
                ])->all(),
                'total' => $sessions->count(),
            ]);
        } catch (\InvalidArgumentException $e) {
            return $this->error($e->getMessage());
        }
    }
}
