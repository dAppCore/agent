<?php

declare(strict_types=1);

namespace Core\Mod\Agentic\Controllers\Api;

use Core\Front\Controller;
use Core\Mod\Agentic\Actions\Brain\ForgetKnowledge;
use Core\Mod\Agentic\Actions\Brain\ListKnowledge;
use Core\Mod\Agentic\Actions\Brain\RecallKnowledge;
use Core\Mod\Agentic\Actions\Brain\RememberKnowledge;
use Illuminate\Http\JsonResponse;
use Illuminate\Http\Request;

class BrainController extends Controller
{
    /**
     * POST /api/brain/remember
     *
     * Store a memory in OpenBrain.
     */
    public function remember(Request $request): JsonResponse
    {
        $validated = $request->validate([
            'content' => 'required|string|max:50000',
            'type' => 'required|string',
            'tags' => 'nullable|array',
            'tags.*' => 'string',
            'project' => 'nullable|string|max:255',
            'confidence' => 'nullable|numeric|min:0|max:1',
            'supersedes' => 'nullable|uuid',
            'expires_in' => 'nullable|integer|min:1',
        ]);

        $workspace = $request->attributes->get('workspace');
        $apiKey = $request->attributes->get('api_key');
        $agentId = $apiKey?->name ?? 'api';

        try {
            $memory = RememberKnowledge::run($validated, $workspace->id, $agentId);

            return response()->json([
                'data' => $memory->toMcpContext(),
            ], 201);
        } catch (\InvalidArgumentException $e) {
            return response()->json([
                'error' => 'validation_error',
                'message' => $e->getMessage(),
            ], 422);
        } catch (\RuntimeException $e) {
            return response()->json([
                'error' => 'service_error',
                'message' => 'Brain service temporarily unavailable.',
            ], 503);
        }
    }

    /**
     * POST /api/brain/recall
     *
     * Semantic search across memories.
     */
    public function recall(Request $request): JsonResponse
    {
        $validated = $request->validate([
            'query' => 'required|string|max:2000',
            'top_k' => 'nullable|integer|min:1|max:20',
            'filter' => 'nullable|array',
            'filter.project' => 'nullable|string',
            'filter.type' => 'nullable',
            'filter.agent_id' => 'nullable|string',
            'filter.min_confidence' => 'nullable|numeric|min:0|max:1',
        ]);

        $workspace = $request->attributes->get('workspace');

        try {
            $result = RecallKnowledge::run(
                $validated['query'],
                $workspace->id,
                $validated['filter'] ?? [],
                $validated['top_k'] ?? 5,
            );

            return response()->json([
                'data' => $result,
            ]);
        } catch (\InvalidArgumentException $e) {
            return response()->json([
                'error' => 'validation_error',
                'message' => $e->getMessage(),
            ], 422);
        } catch (\RuntimeException $e) {
            return response()->json([
                'error' => 'service_error',
                'message' => 'Brain service temporarily unavailable.',
            ], 503);
        }
    }

    /**
     * DELETE /api/brain/forget/{id}
     *
     * Remove a memory.
     */
    public function forget(Request $request, string $id): JsonResponse
    {
        $request->validate([
            'reason' => 'nullable|string|max:500',
        ]);

        $workspace = $request->attributes->get('workspace');
        $apiKey = $request->attributes->get('api_key');
        $agentId = $apiKey?->name ?? 'api';

        try {
            $result = ForgetKnowledge::run($id, $workspace->id, $agentId, $request->input('reason'));

            return response()->json([
                'data' => $result,
            ]);
        } catch (\InvalidArgumentException $e) {
            return response()->json([
                'error' => 'not_found',
                'message' => $e->getMessage(),
            ], 404);
        } catch (\RuntimeException $e) {
            return response()->json([
                'error' => 'service_error',
                'message' => 'Brain service temporarily unavailable.',
            ], 503);
        }
    }

    /**
     * GET /api/brain/list
     *
     * List memories with optional filters.
     */
    public function list(Request $request): JsonResponse
    {
        $validated = $request->validate([
            'project' => 'nullable|string',
            'type' => 'nullable|string',
            'agent_id' => 'nullable|string',
            'limit' => 'nullable|integer|min:1|max:100',
        ]);

        $workspace = $request->attributes->get('workspace');

        try {
            $result = ListKnowledge::run($workspace->id, $validated);

            return response()->json([
                'data' => $result,
            ]);
        } catch (\InvalidArgumentException $e) {
            return response()->json([
                'error' => 'validation_error',
                'message' => $e->getMessage(),
            ], 422);
        }
    }
}
