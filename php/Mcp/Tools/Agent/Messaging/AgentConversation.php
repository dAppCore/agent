<?php

declare(strict_types=1);

namespace Core\Mod\Agentic\Mcp\Tools\Agent\Messaging;

use Core\Mod\Agentic\Mcp\Tools\Agent\AgentTool;
use Core\Mod\Agentic\Models\AgentMessage;

/**
 * View conversation thread between two agents.
 */
class AgentConversation extends AgentTool
{
    protected string $category = 'messaging';

    protected array $scopes = ['read'];

    public function name(): string
    {
        return 'agent_conversation';
    }

    public function description(): string
    {
        return 'View conversation thread with a specific agent. Returns up to 50 messages between you and the target agent.';
    }

    public function inputSchema(): array
    {
        return [
            'type' => 'object',
            'properties' => [
                'me' => [
                    'type' => 'string',
                    'description' => 'Your agent name (e.g. "cladius")',
                    'maxLength' => 100,
                ],
                'agent' => [
                    'type' => 'string',
                    'description' => 'The other agent to view conversation with (e.g. "charon")',
                    'maxLength' => 100,
                ],
            ],
            'required' => ['me', 'agent'],
        ];
    }

    public function handle(array $args, array $context = []): array
    {
        $workspaceId = $context['workspace_id'] ?? null;
        if ($workspaceId === null) {
            return $this->error('workspace_id is required');
        }

        $me = $this->requireString($args, 'me', 100);
        $agent = $this->requireString($args, 'agent', 100);

        $messages = AgentMessage::where('workspace_id', $workspaceId)
            ->conversation($me, $agent)
            ->limit(50)
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
