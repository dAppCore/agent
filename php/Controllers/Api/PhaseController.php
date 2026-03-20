<?php

declare(strict_types=1);

namespace Core\Mod\Agentic\Controllers\Api;

use Core\Front\Controller;
use Core\Mod\Agentic\Actions\Phase\AddCheckpoint;
use Core\Mod\Agentic\Actions\Phase\GetPhase;
use Core\Mod\Agentic\Actions\Phase\UpdatePhaseStatus;
use Illuminate\Http\JsonResponse;
use Illuminate\Http\Request;

class PhaseController extends Controller
{
    /**
     * GET /api/plans/{slug}/phases/{phase}
     */
    public function show(Request $request, string $slug, string $phase): JsonResponse
    {
        $workspace = $request->attributes->get('workspace');

        try {
            $resolved = GetPhase::run($slug, $phase, $workspace->id);

            return response()->json([
                'data' => [
                    'order' => $resolved->order,
                    'name' => $resolved->name,
                    'description' => $resolved->description,
                    'status' => $resolved->status,
                    'tasks' => $resolved->tasks,
                    'checkpoints' => $resolved->getCheckpoints(),
                    'dependencies' => $resolved->dependencies,
                    'task_progress' => $resolved->getTaskProgress(),
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
     * PATCH /api/plans/{slug}/phases/{phase}
     */
    public function update(Request $request, string $slug, string $phase): JsonResponse
    {
        $validated = $request->validate([
            'status' => 'required|string|in:pending,in_progress,completed,blocked,skipped',
            'notes' => 'nullable|string|max:5000',
        ]);

        $workspace = $request->attributes->get('workspace');

        try {
            $resolved = UpdatePhaseStatus::run(
                $slug,
                $phase,
                $validated['status'],
                $workspace->id,
                $validated['notes'] ?? null,
            );

            return response()->json([
                'data' => [
                    'order' => $resolved->order,
                    'name' => $resolved->name,
                    'status' => $resolved->status,
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
     * POST /api/plans/{slug}/phases/{phase}/checkpoint
     */
    public function checkpoint(Request $request, string $slug, string $phase): JsonResponse
    {
        $validated = $request->validate([
            'note' => 'required|string|max:5000',
            'context' => 'nullable|array',
        ]);

        $workspace = $request->attributes->get('workspace');

        try {
            $resolved = AddCheckpoint::run(
                $slug,
                $phase,
                $validated['note'],
                $workspace->id,
                $validated['context'] ?? [],
            );

            return response()->json([
                'data' => [
                    'checkpoints' => $resolved->getCheckpoints(),
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
