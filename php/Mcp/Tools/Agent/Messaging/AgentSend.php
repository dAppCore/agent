<?php

declare(strict_types=1);

namespace Core\Mod\Agentic\Mcp\Tools\Agent\Messaging;

use Core\Mod\Agentic\Mcp\Tools\Agent\AgentTool;
use Core\Mod\Agentic\Models\AgentMessage;

/**
 * Send a direct message to another agent.
 *
 * Chronological, not semantic — messages are stored and retrieved
 * in order, not via vector search.
 */
class AgentSend extends AgentTool
{
    protected string $category = 'messaging';

    protected array $scopes = ['write'];

    public function name(): string
    {
        return 'agent_send';
    }

    public function description(): string
    {
        return 'Send a direct message to another agent. Messages are chronological, not semantic.';
    }

    public function inputSchema(): array
    {
        return [
            'type' => 'object',
            'properties' => [
                'to' => [
                    'type' => 'string',
                    'description' => 'Recipient agent name (e.g. "charon", "cladius")',
                    'maxLength' => 100,
                ],
                'from' => [
                    'type' => 'string',
                    'description' => 'Sender agent name (e.g. "cladius")',
                    'maxLength' => 100,
                ],
                'content' => [
                    'type' => 'string',
                    'description' => 'Message content',
                    'maxLength' => 10000,
                ],
                'subject' => [
                    'type' => 'string',
                    'description' => 'Optional subject line',
                    'maxLength' => 255,
                ],
            ],
            'required' => ['to', 'from', 'content'],
        ];
    }

    public function handle(array $args, array $context = []): array
    {
        $workspaceId = $context['workspace_id'] ?? null;
        if ($workspaceId === null) {
            return $this->error('workspace_id is required');
        }

        $to = $this->requireString($args, 'to', 100);
        $from = $this->requireString($args, 'from', 100);
        $content = $this->requireString($args, 'content', 10000);
        $subject = $this->optionalString($args, 'subject', null, 255);

        $message = AgentMessage::create([
            'workspace_id' => $workspaceId,
            'from_agent' => $from,
            'to_agent' => $to,
            'content' => $content,
            'subject' => $subject,
        ]);

        return $this->success([
            'id' => $message->id,
            'from' => $message->from_agent,
            'to' => $message->to_agent,
            'created_at' => $message->created_at->toIso8601String(),
        ]);
    }
}
