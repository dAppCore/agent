<?php

declare(strict_types=1);

namespace Core\Mod\Agentic\Controllers\Api;

use Core\Mod\Agentic\Models\AgentMessage;
use Illuminate\Http\JsonResponse;
use Illuminate\Http\Request;
use Illuminate\Routing\Controller;

class MessageController extends Controller
{
    /**
     * GET /v1/messages/inbox — unread messages for the requesting agent.
     */
    public function inbox(Request $request): JsonResponse
    {
        $agent = $request->query('agent', $request->header('X-Agent-Name', 'unknown'));
        $workspaceId = $request->attributes->get('workspace_id');

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

        return response()->json(['data' => $messages]);
    }

    /**
     * GET /v1/messages/conversation/{agent} — thread between requesting agent and target.
     */
    public function conversation(Request $request, string $agent): JsonResponse
    {
        $me = $request->query('me', $request->header('X-Agent-Name', 'unknown'));
        $workspaceId = $request->attributes->get('workspace_id');

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

        return response()->json(['data' => $messages]);
    }

    /**
     * POST /v1/messages/send — send a message to another agent.
     */
    public function send(Request $request): JsonResponse
    {
        $validated = $request->validate([
            'to' => 'required|string|max:100',
            'content' => 'required|string|max:10000',
            'from' => 'required|string|max:100',
            'subject' => 'nullable|string|max:255',
        ]);

        $workspaceId = $request->attributes->get('workspace_id');

        $message = AgentMessage::create([
            'workspace_id' => $workspaceId,
            'from_agent' => $validated['from'],
            'to_agent' => $validated['to'],
            'content' => $validated['content'],
            'subject' => $validated['subject'] ?? null,
        ]);

        return response()->json([
            'data' => [
                'id' => $message->id,
                'from' => $message->from_agent,
                'to' => $message->to_agent,
                'created_at' => $message->created_at->toIso8601String(),
            ],
        ], 201);
    }

    /**
     * POST /v1/messages/{id}/read — mark a message as read.
     */
    public function markRead(Request $request, int $id): JsonResponse
    {
        $workspaceId = $request->attributes->get('workspace_id');

        $message = AgentMessage::where('workspace_id', $workspaceId)
            ->findOrFail($id);

        $message->markRead();

        return response()->json(['data' => ['id' => $id, 'read' => true]]);
    }
}
