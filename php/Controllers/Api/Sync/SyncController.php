<?php

// SPDX-License-Identifier: EUPL-1.2

declare(strict_types=1);

namespace Core\Mod\Agentic\Controllers\Api\Sync;

use Core\Front\Controller;
use Core\Mod\Agentic\Actions\Sync\PullFleetContext;
use Core\Mod\Agentic\Actions\Sync\PushDispatchHistory;
use Core\Mod\Agentic\Services\SessionService;
use Illuminate\Http\JsonResponse;
use Illuminate\Http\Request;

class SyncController extends Controller
{
    public function push(Request $request): JsonResponse
    {
        $validated = $request->validate([
            'agent_id' => 'required|string|max:255',
            'dispatches' => 'nullable|array',
            'session_id' => 'nullable|string|max:255',
        ]);

        $result = PushDispatchHistory::run(
            (int) $request->attributes->get('workspace_id'),
            $validated['agent_id'],
            $validated['dispatches'] ?? [],
        );

        $this->emitSessionEvent(
            $validated['session_id'] ?? null,
            'sync.push',
            ['agent_id' => $validated['agent_id'], 'synced' => $result['synced'] ?? 0],
        );

        return response()->json(['data' => $result], 201);
    }

    public function pull(Request $request): JsonResponse
    {
        $validated = $request->validate([
            'agent_id' => 'required|string|max:255',
            'since' => 'nullable|date',
            'session_id' => 'nullable|string|max:255',
        ]);

        $context = PullFleetContext::run(
            (int) $request->attributes->get('workspace_id'),
            $validated['agent_id'],
            $validated['since'] ?? null,
        );

        $this->emitSessionEvent(
            $validated['session_id'] ?? null,
            'sync.pull',
            ['agent_id' => $validated['agent_id'], 'total' => count($context)],
        );

        return response()->json([
            'data' => $context,
            'total' => count($context),
        ]);
    }

    /**
     * @param  array<string, mixed>  $data
     */
    private function emitSessionEvent(?string $sessionId, string $event, array $data): void
    {
        if ($sessionId === null || trim($sessionId) === '') {
            return;
        }

        $service = $this->resolveSessionService();

        if ($service === null || ! method_exists($service, 'emit')) {
            return;
        }

        $service->emit($sessionId, [
            'event' => $event,
            'data' => $data,
        ]);
    }

    private function resolveSessionService(): ?object
    {
        if (! class_exists(SessionService::class)) {
            return null;
        }

        $service = app(SessionService::class);

        return is_object($service) ? $service : null;
    }
}
