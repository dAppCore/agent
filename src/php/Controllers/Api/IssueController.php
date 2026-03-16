<?php

declare(strict_types=1);

namespace Core\Mod\Agentic\Controllers\Api;

use Core\Front\Controller;
use Core\Mod\Agentic\Actions\Issue\AddIssueComment;
use Core\Mod\Agentic\Actions\Issue\ArchiveIssue;
use Core\Mod\Agentic\Actions\Issue\CreateIssue;
use Core\Mod\Agentic\Actions\Issue\GetIssue;
use Core\Mod\Agentic\Actions\Issue\ListIssues;
use Core\Mod\Agentic\Actions\Issue\UpdateIssue;
use Illuminate\Http\JsonResponse;
use Illuminate\Http\Request;

class IssueController extends Controller
{
    /**
     * GET /api/issues
     */
    public function index(Request $request): JsonResponse
    {
        $validated = $request->validate([
            'status' => 'nullable|string|in:open,in_progress,review,closed',
            'type' => 'nullable|string|in:bug,feature,task,improvement',
            'priority' => 'nullable|string|in:low,normal,high,urgent',
            'sprint' => 'nullable|string',
            'label' => 'nullable|string',
            'include_closed' => 'nullable|boolean',
        ]);

        $workspace = $request->attributes->get('workspace');

        try {
            $issues = ListIssues::run(
                $workspace->id,
                $validated['status'] ?? null,
                $validated['type'] ?? null,
                $validated['priority'] ?? null,
                $validated['sprint'] ?? null,
                $validated['label'] ?? null,
                (bool) ($validated['include_closed'] ?? false),
            );

            return response()->json([
                'data' => $issues->map(fn ($issue) => [
                    'slug' => $issue->slug,
                    'title' => $issue->title,
                    'type' => $issue->type,
                    'status' => $issue->status,
                    'priority' => $issue->priority,
                    'assignee' => $issue->assignee,
                    'sprint' => $issue->sprint?->slug,
                    'labels' => $issue->labels ?? [],
                    'updated_at' => $issue->updated_at->toIso8601String(),
                ])->values()->all(),
                'total' => $issues->count(),
            ]);
        } catch (\InvalidArgumentException $e) {
            return response()->json([
                'error' => 'validation_error',
                'message' => $e->getMessage(),
            ], 422);
        }
    }

    /**
     * GET /api/issues/{slug}
     */
    public function show(Request $request, string $slug): JsonResponse
    {
        $workspace = $request->attributes->get('workspace');

        try {
            $issue = GetIssue::run($slug, $workspace->id);

            return response()->json([
                'data' => $issue->toMcpContext(),
            ]);
        } catch (\InvalidArgumentException $e) {
            return response()->json([
                'error' => 'not_found',
                'message' => $e->getMessage(),
            ], 404);
        }
    }

    /**
     * POST /api/issues
     */
    public function store(Request $request): JsonResponse
    {
        $validated = $request->validate([
            'title' => 'required|string|max:255',
            'slug' => 'nullable|string|max:255',
            'description' => 'nullable|string|max:10000',
            'type' => 'nullable|string|in:bug,feature,task,improvement',
            'priority' => 'nullable|string|in:low,normal,high,urgent',
            'labels' => 'nullable|array',
            'labels.*' => 'string',
            'assignee' => 'nullable|string|max:255',
            'reporter' => 'nullable|string|max:255',
            'sprint_id' => 'nullable|integer|exists:sprints,id',
            'metadata' => 'nullable|array',
        ]);

        $workspace = $request->attributes->get('workspace');

        try {
            $issue = CreateIssue::run($validated, $workspace->id);

            return response()->json([
                'data' => [
                    'slug' => $issue->slug,
                    'title' => $issue->title,
                    'type' => $issue->type,
                    'status' => $issue->status,
                    'priority' => $issue->priority,
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
     * PATCH /api/issues/{slug}
     */
    public function update(Request $request, string $slug): JsonResponse
    {
        $validated = $request->validate([
            'status' => 'nullable|string|in:open,in_progress,review,closed',
            'priority' => 'nullable|string|in:low,normal,high,urgent',
            'type' => 'nullable|string|in:bug,feature,task,improvement',
            'title' => 'nullable|string|max:255',
            'description' => 'nullable|string|max:10000',
            'assignee' => 'nullable|string|max:255',
            'sprint_id' => 'nullable|integer|exists:sprints,id',
            'labels' => 'nullable|array',
            'labels.*' => 'string',
        ]);

        $workspace = $request->attributes->get('workspace');

        try {
            $issue = UpdateIssue::run($slug, $validated, $workspace->id);

            return response()->json([
                'data' => [
                    'slug' => $issue->slug,
                    'title' => $issue->title,
                    'status' => $issue->status,
                    'priority' => $issue->priority,
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
     * DELETE /api/issues/{slug}
     */
    public function destroy(Request $request, string $slug): JsonResponse
    {
        $request->validate([
            'reason' => 'nullable|string|max:500',
        ]);

        $workspace = $request->attributes->get('workspace');

        try {
            $issue = ArchiveIssue::run($slug, $workspace->id, $request->input('reason'));

            return response()->json([
                'data' => [
                    'slug' => $issue->slug,
                    'status' => $issue->status,
                    'archived_at' => $issue->archived_at?->toIso8601String(),
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
     * GET /api/issues/{slug}/comments
     */
    public function comments(Request $request, string $slug): JsonResponse
    {
        $workspace = $request->attributes->get('workspace');

        try {
            $issue = GetIssue::run($slug, $workspace->id);
            $comments = $issue->comments;

            return response()->json([
                'data' => $comments->map(fn ($c) => $c->toMcpContext())->values()->all(),
                'total' => $comments->count(),
            ]);
        } catch (\InvalidArgumentException $e) {
            return response()->json([
                'error' => 'not_found',
                'message' => $e->getMessage(),
            ], 404);
        }
    }

    /**
     * POST /api/issues/{slug}/comments
     */
    public function addComment(Request $request, string $slug): JsonResponse
    {
        $validated = $request->validate([
            'author' => 'required|string|max:255',
            'body' => 'required|string|max:10000',
            'metadata' => 'nullable|array',
        ]);

        $workspace = $request->attributes->get('workspace');

        try {
            $comment = AddIssueComment::run(
                $slug,
                $workspace->id,
                $validated['author'],
                $validated['body'],
                $validated['metadata'] ?? null,
            );

            return response()->json([
                'data' => $comment->toMcpContext(),
            ], 201);
        } catch (\InvalidArgumentException $e) {
            return response()->json([
                'error' => 'not_found',
                'message' => $e->getMessage(),
            ], 404);
        }
    }
}
