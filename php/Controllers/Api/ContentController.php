<?php

declare(strict_types=1);

namespace Core\Mod\Agentic\Controllers\Api;

use Core\Front\Controller;
use Core\Mod\Content\Models\ContentBrief;
use Illuminate\Http\JsonResponse;
use Illuminate\Http\Request;
use Illuminate\Support\Str;

/**
 * Content API — reusable content briefs.
 *
 * The agent's content pipeline (pkg/agentic/content.go) creates, reads, and
 * lists briefs that later drive generation. Briefs persist to the php-content
 * `content_briefs` table.
 *
 * NOTE: that table's live schema (title, description, type, uuid, …) does not
 * carry the agent contract's slug/product/category/summary/metadata, so those
 * fields are accepted but not persisted (decision: map onto existing columns,
 * no schema change). The brief's `uuid` doubles as the stable external slug.
 */
class ContentController extends Controller
{
    /**
     * GET /v1/content/briefs — list briefs, newest first.
     */
    public function index(Request $request): JsonResponse
    {
        $validated = $request->validate([
            // category/product are accepted for contract parity but not stored,
            // so they cannot be filtered on; limit is honoured.
            'category' => 'nullable|string|max:120',
            'product' => 'nullable|string|max:120',
            'limit' => 'nullable|integer|min:1|max:200',
        ]);

        $briefs = ContentBrief::query()
            ->where('workspace_id', $this->workspaceId($request))
            ->orderByDesc('created_at')
            ->limit((int) ($validated['limit'] ?? 50))
            ->get();

        return response()->json([
            'data' => [
                'briefs' => $briefs->map(fn (ContentBrief $b) => $this->briefResource($b))->all(),
                'total' => $briefs->count(),
            ],
        ]);
    }

    /**
     * POST /v1/content/briefs — create a brief.
     */
    public function store(Request $request): JsonResponse
    {
        $validated = $request->validate([
            'title' => 'nullable|string|max:255',
            'name' => 'nullable|string|max:255',
            'brief' => 'nullable|string',
            'content_type' => 'nullable|string|max:32',
            // accepted, not persisted (no column): slug, product, category, summary, metadata
        ]);

        $title = $validated['title'] ?? $validated['name'] ?? null;

        if ($title === null && empty($validated['brief'])) {
            return response()->json([
                'error' => 'validation_error',
                'message' => 'A title (or name) or brief is required',
            ], 422);
        }

        $title ??= Str::limit((string) $validated['brief'], 80, '');

        $brief = new ContentBrief;
        $brief->forceFill([
            'uuid' => (string) Str::uuid(),
            'workspace_id' => $this->workspaceId($request),
            'user_id' => $this->userId($request),
            'title' => $title,
            'description' => $validated['brief'] ?? null,
            'type' => $validated['content_type'] ?? 'article',
            'status' => 'draft',
        ])->save();

        return response()->json(['data' => ['brief' => $this->briefResource($brief)]], 201);
    }

    /**
     * GET /v1/content/briefs/{id} — fetch a brief by numeric id or uuid (slug).
     */
    public function show(Request $request, string $id): JsonResponse
    {
        $brief = ContentBrief::query()
            ->where('workspace_id', $this->workspaceId($request))
            ->where(function ($q) use ($id) {
                $q->where('uuid', $id);
                if (ctype_digit($id)) {
                    $q->orWhere('id', (int) $id);
                }
            })
            ->first();

        if ($brief === null) {
            return response()->json([
                'error' => 'not_found',
                'message' => "Brief not found: {$id}",
            ], 404);
        }

        return response()->json(['data' => ['brief' => $this->briefResource($brief)]]);
    }

    /**
     * Serialise a brief into the Go ContentBrief shape. uuid stands in for slug;
     * product/category/summary/metadata have no column and come back empty.
     *
     * @return array<string, mixed>
     */
    private function briefResource(ContentBrief $brief): array
    {
        return [
            'id' => (string) $brief->id,
            'slug' => $brief->uuid,
            'name' => $brief->title,
            'title' => $brief->title,
            'brief' => $brief->description,
            'created_at' => $brief->created_at?->toIso8601String(),
            'updated_at' => $brief->updated_at?->toIso8601String(),
        ];
    }

    private function workspaceId(Request $request): ?int
    {
        $id = $request->attributes->get('workspace_id');

        return $id !== null ? (int) $id : null;
    }

    private function userId(Request $request): ?int
    {
        $apiKey = $request->attributes->get('api_key');

        return $apiKey?->user_id ?? $request->attributes->get('user_id');
    }
}
