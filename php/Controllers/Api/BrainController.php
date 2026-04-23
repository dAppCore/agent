<?php

declare(strict_types=1);

namespace Core\Mod\Agentic\Controllers\Api;

use Core\Front\Controller;
use Core\Mod\Agentic\Actions\Brain\ForgetKnowledge;
use Core\Mod\Agentic\Actions\Brain\ListKnowledge;
use Core\Mod\Agentic\Actions\Brain\RememberKnowledge;
use Core\Mod\Agentic\Models\BrainMemory;
use Core\Mod\Agentic\Services\BrainService;
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
        $workspaceId = (int) ($request->attributes->get('workspace_id') ?? $workspace?->id);
        $apiKey = $request->attributes->get('api_key') ?? $request->attributes->get('agent_api_key');
        $agentId = $apiKey?->name ?? 'api';

        try {
            $memory = RememberKnowledge::run($validated, $workspaceId, $agentId);

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
    public function recall(Request $request, BrainService $brain): JsonResponse
    {
        $validated = $request->validate([
            'query' => 'required|string|max:2000',
            'limit' => 'nullable|integer|min:1|max:20',
            'top_k' => 'nullable|integer|min:1|max:20',
            'workspace_id' => 'nullable|integer|min:1',
            'org' => 'nullable|string',
            'project' => 'nullable|string',
            'type' => 'nullable',
            'keywords' => 'nullable|array',
            'keywords.*' => 'string|max:255',
            'boost_keywords' => 'nullable|array',
            'boost_keywords.*' => 'string|max:255',
            'filter' => 'nullable|array',
            'filter.org' => 'nullable|string',
            'filter.project' => 'nullable|string',
            'filter.type' => 'nullable',
            'filter.agent_id' => 'nullable|string',
            'filter.min_confidence' => 'nullable|numeric|min:0|max:1',
        ]);

        $workspace = $request->attributes->get('workspace');
        $workspaceId = (int) ($request->attributes->get('workspace_id') ?? $workspace?->id ?? $validated['workspace_id'] ?? 0);
        $filter = $validated['filter'] ?? [];

        foreach (['org', 'project', 'type'] as $field) {
            if (array_key_exists($field, $validated) && $validated[$field] !== null) {
                $filter[$field] = $validated[$field];
            }
        }

        try {
            $this->assertValidTypeFilter($filter['type'] ?? null);

            $result = $brain->recall(
                $validated['query'],
                $validated['limit'] ?? $validated['top_k'] ?? 5,
                $filter,
                $workspaceId,
                $this->normaliseStringList($validated['keywords'] ?? []),
                $this->normaliseStringList($validated['boost_keywords'] ?? []),
            );
            $result['count'] = count($result['memories'] ?? []);

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
     * @param  array<int, mixed>  $values
     * @return array<int, string>
     */
    private function normaliseStringList(array $values): array
    {
        return array_values(array_filter(array_map(
            static fn (mixed $value): string => is_string($value) ? trim($value) : '',
            $values,
        ), static fn (string $value): bool => $value !== ''));
    }

    private function assertValidTypeFilter(mixed $type): void
    {
        if ($type === null) {
            return;
        }

        $validTypes = BrainMemory::VALID_TYPES;

        if (is_string($type)) {
            if (! in_array($type, $validTypes, true)) {
                throw new \InvalidArgumentException(
                    sprintf('filter.type must be one of: %s', implode(', ', $validTypes))
                );
            }

            return;
        }

        if (is_array($type)) {
            foreach ($type as $value) {
                if (! is_string($value) || ! in_array($value, $validTypes, true)) {
                    throw new \InvalidArgumentException(
                        sprintf('Each filter.type value must be one of: %s', implode(', ', $validTypes))
                    );
                }
            }

            return;
        }

        throw new \InvalidArgumentException(
            sprintf('filter.type must be one of: %s', implode(', ', $validTypes))
        );
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
        $workspaceId = (int) ($request->attributes->get('workspace_id') ?? $workspace?->id);
        $apiKey = $request->attributes->get('api_key') ?? $request->attributes->get('agent_api_key');
        $agentId = $apiKey?->name ?? 'api';

        try {
            $result = ForgetKnowledge::run($id, $workspaceId, $agentId, $request->input('reason'));

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
        $workspaceId = (int) ($request->attributes->get('workspace_id') ?? $workspace?->id);

        try {
            $result = ListKnowledge::run($workspaceId, $validated);

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

    /**
     * GET /v1/brain/tags
     *
     * List distinct memory tags and document counts.
     */
    public function tags(BrainService $brain): JsonResponse
    {
        try {
            $result = $brain->elasticAggregate([
                'size' => 0,
                'aggs' => [
                    'tags' => [
                        'terms' => [
                            'field' => 'tags.keyword',
                            'size' => 1000,
                        ],
                    ],
                ],
            ]);

            $tags = [];
            $buckets = $result['aggregations']['tags']['buckets'] ?? [];

            if (is_array($buckets)) {
                foreach ($buckets as $bucket) {
                    if (! is_array($bucket) || ! is_string($bucket['key'] ?? null)) {
                        continue;
                    }

                    $tags[$bucket['key']] = (int) ($bucket['doc_count'] ?? 0);
                }
            }

            return response()->json([
                'data' => $tags,
            ]);
        } catch (\RuntimeException $e) {
            return response()->json([
                'error' => 'service_error',
                'message' => 'Brain service temporarily unavailable.',
            ], 503);
        }
    }

    /**
     * GET /v1/brain/scopes
     *
     * List distinct organisation/project memory scopes.
     */
    public function scopes(BrainService $brain): JsonResponse
    {
        try {
            $result = $brain->elasticAggregate([
                'size' => 0,
                'aggs' => [
                    'scopes' => [
                        'composite' => [
                            'size' => 1000,
                            'sources' => [
                                [
                                    'org' => [
                                        'terms' => [
                                            'field' => 'org.keyword',
                                        ],
                                    ],
                                ],
                                [
                                    'project' => [
                                        'terms' => [
                                            'field' => 'project.keyword',
                                        ],
                                    ],
                                ],
                            ],
                        ],
                    ],
                ],
            ]);

            $scopes = [];
            $buckets = $result['aggregations']['scopes']['buckets'] ?? [];

            if (is_array($buckets)) {
                foreach ($buckets as $bucket) {
                    $key = is_array($bucket) ? ($bucket['key'] ?? null) : null;

                    if (! is_array($key) || ! is_string($key['org'] ?? null) || ! is_string($key['project'] ?? null)) {
                        continue;
                    }

                    $scopes[$key['org']][$key['project']] = (int) ($bucket['doc_count'] ?? 0);
                }
            }

            return response()->json([
                'data' => $scopes,
            ]);
        } catch (\RuntimeException $e) {
            return response()->json([
                'error' => 'service_error',
                'message' => 'Brain service temporarily unavailable.',
            ], 503);
        }
    }
}
