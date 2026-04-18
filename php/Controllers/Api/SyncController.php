<?php

declare(strict_types=1);

namespace Core\Mod\Agentic\Controllers\Api;

use Core\Front\Controller;
use Core\Mod\Agentic\Actions\Sync\GetAgentSyncStatus;
use Core\Mod\Agentic\Actions\Sync\PullFleetContext;
use Core\Mod\Agentic\Actions\Sync\PushDispatchHistory;
use Illuminate\Http\JsonResponse;
use Illuminate\Http\Request;

class SyncController extends Controller
{
    public function push(Request $request): JsonResponse
    {
        $validated = $request->validate([
            'agent_id' => 'required|string|max:255',
            'dispatches' => 'nullable|array',
        ]);

        $result = PushDispatchHistory::run(
            (int) $request->attributes->get('workspace_id'),
            $validated['agent_id'],
            $validated['dispatches'] ?? [],
        );

        return response()->json(['data' => $result], 201);
    }

    public function pull(Request $request): JsonResponse
    {
        $validated = $request->validate([
            'agent_id' => 'required|string|max:255',
            'since' => 'nullable|date',
        ]);

        $context = PullFleetContext::run(
            (int) $request->attributes->get('workspace_id'),
            $validated['agent_id'],
            $validated['since'] ?? null,
        );

        return response()->json([
            'data' => $context,
            'total' => count($context),
        ]);
    }

    public function status(Request $request): JsonResponse
    {
        $validated = $request->validate([
            'agent_id' => 'required|string|max:255',
        ]);

        $status = GetAgentSyncStatus::run(
            (int) $request->attributes->get('workspace_id'),
            $validated['agent_id'],
        );

        return response()->json(['data' => $status]);
    }
}
