<?php

declare(strict_types=1);

namespace Core\Mod\Agentic\Controllers\Api;

use Core\Front\Controller;
use Core\Mod\Agentic\Actions\Plan\ArchivePlan;
use Core\Mod\Agentic\Actions\Plan\CreatePlan;
use Core\Mod\Agentic\Actions\Plan\GetPlan;
use Core\Mod\Agentic\Actions\Plan\ListPlans;
use Core\Mod\Agentic\Actions\Plan\UpdatePlanStatus;
use Illuminate\Http\JsonResponse;
use Illuminate\Http\Request;

class PlanController extends Controller
{
    /**
     * GET /api/plans
     */
    public function index(Request $request): JsonResponse
    {
        $validated = $request->validate([
            'status' => 'nullable|string|in:draft,active,paused,completed,archived',
            'include_archived' => 'nullable|boolean',
        ]);

        $workspace = $request->attributes->get('workspace');

        try {
            $plans = ListPlans::run(
                $workspace->id,
                $validated['status'] ?? null,
                (bool) ($validated['include_archived'] ?? false),
            );

            return response()->json([
                'data' => $plans->map(fn ($plan) => [
                    'slug' => $plan->slug,
                    'title' => $plan->title,
                    'status' => $plan->status,
                    'progress' => $plan->getProgress(),
                    'updated_at' => $plan->updated_at->toIso8601String(),
                ])->values()->all(),
                'total' => $plans->count(),
            ]);
        } catch (\InvalidArgumentException $e) {
            return response()->json([
                'error' => 'validation_error',
                'message' => $e->getMessage(),
            ], 422);
        }
    }

    /**
     * GET /api/plans/{slug}
     */
    public function show(Request $request, string $slug): JsonResponse
    {
        $workspace = $request->attributes->get('workspace');

        try {
            $plan = GetPlan::run($slug, $workspace->id);

            return response()->json([
                'data' => $plan->toMcpContext(),
            ]);
        } catch (\InvalidArgumentException $e) {
            return response()->json([
                'error' => 'not_found',
                'message' => $e->getMessage(),
            ], 404);
        }
    }

    /**
     * POST /api/plans
     */
    public function store(Request $request): JsonResponse
    {
        $validated = $request->validate([
            'title' => 'required|string|max:255',
            'slug' => 'nullable|string|max:255',
            'description' => 'nullable|string|max:10000',
            'context' => 'nullable|array',
            'phases' => 'nullable|array',
            'phases.*.name' => 'required_with:phases|string',
            'phases.*.description' => 'nullable|string',
            'phases.*.tasks' => 'nullable|array',
            'phases.*.tasks.*' => 'string',
        ]);

        $workspace = $request->attributes->get('workspace');

        try {
            $plan = CreatePlan::run($validated, $workspace->id);

            return response()->json([
                'data' => [
                    'slug' => $plan->slug,
                    'title' => $plan->title,
                    'status' => $plan->status,
                    'phases' => $plan->agentPhases->count(),
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
     * PATCH /api/plans/{slug}
     */
    public function update(Request $request, string $slug): JsonResponse
    {
        $validated = $request->validate([
            'status' => 'required|string|in:draft,active,paused,completed',
        ]);

        $workspace = $request->attributes->get('workspace');

        try {
            $plan = UpdatePlanStatus::run($slug, $validated['status'], $workspace->id);

            return response()->json([
                'data' => [
                    'slug' => $plan->slug,
                    'status' => $plan->status,
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
     * DELETE /api/plans/{slug}
     */
    public function destroy(Request $request, string $slug): JsonResponse
    {
        $request->validate([
            'reason' => 'nullable|string|max:500',
        ]);

        $workspace = $request->attributes->get('workspace');

        try {
            $plan = ArchivePlan::run($slug, $workspace->id, $request->input('reason'));

            return response()->json([
                'data' => [
                    'slug' => $plan->slug,
                    'status' => 'archived',
                    'archived_at' => $plan->archived_at?->toIso8601String(),
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
