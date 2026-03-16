<?php

declare(strict_types=1);

namespace Core\Mod\Agentic\Controllers\Api;

use Core\Front\Controller;
use Core\Mod\Agentic\Actions\Sprint\ArchiveSprint;
use Core\Mod\Agentic\Actions\Sprint\CreateSprint;
use Core\Mod\Agentic\Actions\Sprint\GetSprint;
use Core\Mod\Agentic\Actions\Sprint\ListSprints;
use Core\Mod\Agentic\Actions\Sprint\UpdateSprint;
use Illuminate\Http\JsonResponse;
use Illuminate\Http\Request;

class SprintController extends Controller
{
    /**
     * GET /api/sprints
     */
    public function index(Request $request): JsonResponse
    {
        $validated = $request->validate([
            'status' => 'nullable|string|in:planning,active,completed,cancelled',
            'include_cancelled' => 'nullable|boolean',
        ]);

        $workspace = $request->attributes->get('workspace');

        try {
            $sprints = ListSprints::run(
                $workspace->id,
                $validated['status'] ?? null,
                (bool) ($validated['include_cancelled'] ?? false),
            );

            return response()->json([
                'data' => $sprints->map(fn ($sprint) => [
                    'slug' => $sprint->slug,
                    'title' => $sprint->title,
                    'status' => $sprint->status,
                    'progress' => $sprint->getProgress(),
                    'started_at' => $sprint->started_at?->toIso8601String(),
                    'ended_at' => $sprint->ended_at?->toIso8601String(),
                    'updated_at' => $sprint->updated_at->toIso8601String(),
                ])->values()->all(),
                'total' => $sprints->count(),
            ]);
        } catch (\InvalidArgumentException $e) {
            return response()->json([
                'error' => 'validation_error',
                'message' => $e->getMessage(),
            ], 422);
        }
    }

    /**
     * GET /api/sprints/{slug}
     */
    public function show(Request $request, string $slug): JsonResponse
    {
        $workspace = $request->attributes->get('workspace');

        try {
            $sprint = GetSprint::run($slug, $workspace->id);

            return response()->json([
                'data' => $sprint->toMcpContext(),
            ]);
        } catch (\InvalidArgumentException $e) {
            return response()->json([
                'error' => 'not_found',
                'message' => $e->getMessage(),
            ], 404);
        }
    }

    /**
     * POST /api/sprints
     */
    public function store(Request $request): JsonResponse
    {
        $validated = $request->validate([
            'title' => 'required|string|max:255',
            'slug' => 'nullable|string|max:255',
            'description' => 'nullable|string|max:10000',
            'goal' => 'nullable|string|max:10000',
            'metadata' => 'nullable|array',
        ]);

        $workspace = $request->attributes->get('workspace');

        try {
            $sprint = CreateSprint::run($validated, $workspace->id);

            return response()->json([
                'data' => [
                    'slug' => $sprint->slug,
                    'title' => $sprint->title,
                    'status' => $sprint->status,
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
     * PATCH /api/sprints/{slug}
     */
    public function update(Request $request, string $slug): JsonResponse
    {
        $validated = $request->validate([
            'status' => 'nullable|string|in:planning,active,completed,cancelled',
            'title' => 'nullable|string|max:255',
            'description' => 'nullable|string|max:10000',
            'goal' => 'nullable|string|max:10000',
        ]);

        $workspace = $request->attributes->get('workspace');

        try {
            $sprint = UpdateSprint::run($slug, $validated, $workspace->id);

            return response()->json([
                'data' => [
                    'slug' => $sprint->slug,
                    'title' => $sprint->title,
                    'status' => $sprint->status,
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
     * DELETE /api/sprints/{slug}
     */
    public function destroy(Request $request, string $slug): JsonResponse
    {
        $request->validate([
            'reason' => 'nullable|string|max:500',
        ]);

        $workspace = $request->attributes->get('workspace');

        try {
            $sprint = ArchiveSprint::run($slug, $workspace->id, $request->input('reason'));

            return response()->json([
                'data' => [
                    'slug' => $sprint->slug,
                    'status' => $sprint->status,
                    'archived_at' => $sprint->archived_at?->toIso8601String(),
                ],
            ]);
        } catch (\InvalidArgumentException $e) {
            return response()->json([
                'error' => 'not_found',
                'message' => $e->getMessage(),
            ], 404);
        }
    }
}
