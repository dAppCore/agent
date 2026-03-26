<?php

declare(strict_types=1);

namespace Core\Mod\Agentic\Mcp\Tools\Agent\Messaging;

use Core\Mod\Agentic\Mcp\Tools\Agent\AgentTool;
use Core\Mod\Agentic\Models\AgentMessage;

/**
 * Check inbox — latest messages sent to the requesting agent.
 */
class AgentInbox extends AgentTool
{
    protected string $category = 'messaging';

    protected array $scopes = ['read'];

    public function name(): string
    {
        return 'agent_inbox';
    }

    public function description(): string
    {
        return 'Check your inbox — latest messages sent to you. Returns up to 20 most recent messages.';
    }

    public function inputSchema(): array
    {
        return [
            'type' => 'object',
            'properties' => [
                'agent' => [
                    'type' => 'string',
                    'description' => 'Your agent name (e.g. "cladius", "charon")',
                    'maxLength' => 100,
                ],
            ],
            'required' => ['agent'],
        ];
    }

    public function handle(array $args, array $context = []): array
    {
        $workspaceId = $context['workspace_id'] ?? null;
        if ($workspaceId === null) {
            return $this->error('workspace_id is required');
        }

        $agent = $this->requireString($args, 'agent', 100);

        $messages = AgentMessage::where('workspace_id', $workspaceId)
            ->inbox($agent)
            ->limit(20)
            ->get()
            ->map(fn (AgentMessage $m) => [
                'id' => $m->id,
                'from' => $m->from_agent,
                'to' => $m->to_agent,
                'subject' => $m->subject,
                'content' => $m->content,
                'read' => $m->read_at !== null,
                'created_at' => $m->created_at->toIso8601String(),
            ]);

        return $this->success([
            'count' => $messages->count(),
            'messages' => $messages->toArray(),
        ]);
    }
}
