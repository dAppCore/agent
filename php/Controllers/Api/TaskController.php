<?php

declare(strict_types=1);

namespace Core\Mod\Agentic\Controllers\Api;

use Core\Front\Controller;
use Core\Mod\Agentic\Actions\Task\ToggleTask;
use Core\Mod\Agentic\Actions\Task\UpdateTask;
use Illuminate\Http\JsonResponse;
use Illuminate\Http\Request;

class TaskController extends Controller
{
    /**
     * PATCH /api/plans/{slug}/phases/{phase}/tasks/{index}
     */
    public function update(Request $request, string $slug, string $phase, int $index): JsonResponse
    {
        $validated = $request->validate([
            'status' => 'nullable|string|in:pending,in_progress,completed,blocked,skipped',
            'notes' => 'nullable|string|max:5000',
        ]);

        $workspace = $request->attributes->get('workspace');

        try {
            $result = UpdateTask::run(
                $slug,
                $phase,
                $index,
                $workspace->id,
                $validated['status'] ?? null,
                $validated['notes'] ?? null,
            );

            return response()->json([
                'data' => $result,
            ]);
        } catch (\InvalidArgumentException $e) {
            return response()->json([
                'error' => 'not_found',
                'message' => $e->getMessage(),
            ], 404);
        }
    }

    /**
     * POST /api/plans/{slug}/phases/{phase}/tasks/{index}/toggle
     */
    public function toggle(Request $request, string $slug, string $phase, int $index): JsonResponse
    {
        $workspace = $request->attributes->get('workspace');

        try {
            $result = ToggleTask::run($slug, $phase, $index, $workspace->id);

            return response()->json([
                'data' => $result,
            ]);
        } catch (\InvalidArgumentException $e) {
            return response()->json([
                'error' => 'not_found',
                'message' => $e->getMessage(),
            ], 404);
        }
    }
}
