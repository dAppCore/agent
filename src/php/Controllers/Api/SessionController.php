<?php

declare(strict_types=1);

namespace Core\Mod\Agentic\Controllers\Api;

use Core\Front\Controller;
use Core\Mod\Agentic\Actions\Session\ContinueSession;
use Core\Mod\Agentic\Actions\Session\EndSession;
use Core\Mod\Agentic\Actions\Session\GetSession;
use Core\Mod\Agentic\Actions\Session\ListSessions;
use Core\Mod\Agentic\Actions\Session\StartSession;
use Illuminate\Http\JsonResponse;
use Illuminate\Http\Request;

class SessionController extends Controller
{
    /**
     * GET /api/sessions
     */
    public function index(Request $request): JsonResponse
    {
        $validated = $request->validate([
            'status' => 'nullable|string|in:active,paused,completed,failed',
            'plan_slug' => 'nullable|string|max:255',
            'limit' => 'nullable|integer|min:1|max:1000',
        ]);

        $workspace = $request->attributes->get('workspace');

        try {
            $sessions = ListSessions::run(
                $workspace->id,
                $validated['status'] ?? null,
                $validated['plan_slug'] ?? null,
                $validated['limit'] ?? null,
            );

            return response()->json([
                'data' => $sessions->map(fn ($session) => [
                    'session_id' => $session->session_id,
                    'agent_type' => $session->agent_type,
                    'status' => $session->status,
                    'plan' => $session->plan?->slug,
                    'duration' => $session->getDurationFormatted(),
                    'started_at' => $session->started_at->toIso8601String(),
                    'last_active_at' => $session->last_active_at->toIso8601String(),
                ])->values()->all(),
                'total' => $sessions->count(),
            ]);
        } catch (\InvalidArgumentException $e) {
            return response()->json([
                'error' => 'validation_error',
                'message' => $e->getMessage(),
            ], 422);
        }
    }

    /**
     * GET /api/sessions/{id}
     */
    public function show(Request $request, string $id): JsonResponse
    {
        $workspace = $request->attributes->get('workspace');

        try {
            $session = GetSession::run($id, $workspace->id);

            return response()->json([
                'data' => $session->toMcpContext(),
            ]);
        } catch (\InvalidArgumentException $e) {
            return response()->json([
                'error' => 'not_found',
                'message' => $e->getMessage(),
            ], 404);
        }
    }

    /**
     * POST /api/sessions
     */
    public function store(Request $request): JsonResponse
    {
        $validated = $request->validate([
            'agent_type' => 'required|string|max:50',
            'plan_slug' => 'nullable|string|max:255',
            'context' => 'nullable|array',
        ]);

        $workspace = $request->attributes->get('workspace');

        try {
            $session = StartSession::run(
                $validated['agent_type'],
                $validated['plan_slug'] ?? null,
                $workspace->id,
                $validated['context'] ?? [],
            );

            return response()->json([
                'data' => [
                    'session_id' => $session->session_id,
                    'agent_type' => $session->agent_type,
                    'plan' => $session->plan?->slug,
                    'status' => $session->status,
                ],
            ], 201);
        } catch (\InvalidArgumentException $e) {
            return response()->json([
                'error' => 'validation_error',
                'message' => $e->getMessage(),
            ], 422);
        }
    }

    /**
     * POST /api/sessions/{id}/end
     */
    public function end(Request $request, string $id): JsonResponse
    {
        $validated = $request->validate([
            'status' => 'required|string|in:completed,handed_off,paused,failed',
            'summary' => 'nullable|string|max:10000',
        ]);

        try {
            $session = EndSession::run($id, $validated['status'], $validated['summary'] ?? null);

            return response()->json([
                'data' => [
                    'session_id' => $session->session_id,
                    'status' => $session->status,
                    'duration' => $session->getDurationFormatted(),
                ],
            ]);
        } catch (\InvalidArgumentException $e) {
            return response()->json([
                'error' => 'not_found',
                'message' => $e->getMessage(),
            ], 404);
        }
    }

    /**
     * POST /api/sessions/{id}/continue
     */
    public function continue(Request $request, string $id): JsonResponse
    {
        $validated = $request->validate([
            'agent_type' => 'required|string|max:50',
        ]);

        try {
            $session = ContinueSession::run($id, $validated['agent_type']);

            return response()->json([
                'data' => [
                    'session_id' => $session->session_id,
                    'agent_type' => $session->agent_type,
                    'plan' => $session->plan?->slug,
                    'status' => $session->status,
                    'continued_from' => $session->context_summary['continued_from'] ?? null,
                ],
            ], 201);
        } catch (\InvalidArgumentException $e) {
            return response()->json([
                'error' => 'not_found',
                'message' => $e->getMessage(),
            ], 404);
        }
    }
}
