<?php

declare(strict_types=1);

namespace Core\Mod\Agentic\Services;

use Core\Mod\Agentic\Models\BrainMemory;
use Illuminate\Support\Facades\DB;
use Illuminate\Support\Facades\Http;
use Illuminate\Support\Facades\Log;

class BrainService
{
    private const DEFAULT_MODEL = 'embeddinggemma';

    private const VECTOR_DIMENSION = 768;

    public function __construct(
        private string $ollamaUrl = 'http://localhost:11434',
        private string $qdrantUrl = 'http://localhost:6334',
        private string $collection = 'openbrain',
        private string $embeddingModel = self::DEFAULT_MODEL,
        private bool $verifySsl = true,
    ) {}

    /**
     * Create an HTTP client with common settings.
     */
    private function http(int $timeout = 10): \Illuminate\Http\Client\PendingRequest
    {
        return $this->verifySsl
            ? Http::timeout($timeout)
            : Http::withoutVerifying()->timeout($timeout);
    }

    /**
     * Generate an embedding vector for the given text.
     *
     * @return array<float>
     *
     * @throws \RuntimeException
     */
    public function embed(string $text): array
    {
        $response = $this->http(30)
            ->post("{$this->ollamaUrl}/api/embeddings", [
                'model' => $this->embeddingModel,
                'prompt' => $text,
            ]);

        if (! $response->successful()) {
            throw new \RuntimeException("Ollama embedding failed: {$response->status()}");
        }

        $embedding = $response->json('embedding');

        if (! is_array($embedding) || empty($embedding)) {
            throw new \RuntimeException('Ollama returned no embedding vector');
        }

        return $embedding;
    }

    /**
     * Store a memory in both MariaDB and Qdrant.
     *
     * Creates the MariaDB record and upserts the Qdrant vector within
     * a single DB transaction. If the memory supersedes an older one,
     * the old entry is soft-deleted from MariaDB and removed from Qdrant.
     *
     * @param  array<string, mixed>  $attributes  Fillable attributes for BrainMemory
     * @return BrainMemory The created memory
     */
    public function remember(array $attributes): BrainMemory
    {
        $vector = $this->embed($attributes['content']);

        return DB::connection('brain')->transaction(function () use ($attributes, $vector) {
            $memory = BrainMemory::create($attributes);

            $payload = $this->buildQdrantPayload($memory->id, [
                'workspace_id' => $memory->workspace_id,
                'agent_id' => $memory->agent_id,
                'type' => $memory->type,
                'tags' => $memory->tags ?? [],
                'project' => $memory->project,
                'confidence' => $memory->confidence,
                'source' => $memory->source ?? 'manual',
                'created_at' => $memory->created_at->toIso8601String(),
            ]);
            $payload['vector'] = $vector;

            $this->qdrantUpsert([$payload]);

            if ($memory->supersedes_id) {
                BrainMemory::where('id', $memory->supersedes_id)->delete();
                $this->qdrantDelete([$memory->supersedes_id]);
            }

            return $memory;
        });
    }

    /**
     * Semantic search: find memories similar to the query.
     *
     * @param  array<string, mixed>  $filter  Optional filter criteria
     * @return array{memories: array, scores: array<string, float>}
     */
    public function recall(string $query, int $topK, array $filter, int $workspaceId): array
    {
        $vector = $this->embed($query);

        $filter['workspace_id'] = $workspaceId;
        $qdrantFilter = $this->buildQdrantFilter($filter);

        $response = $this->http(10)
            ->post("{$this->qdrantUrl}/collections/{$this->collection}/points/search", [
                'vector' => $vector,
                'filter' => $qdrantFilter,
                'limit' => $topK,
                'with_payload' => false,
            ]);

        if (! $response->successful()) {
            throw new \RuntimeException("Qdrant search failed: {$response->status()}");
        }

        $results = $response->json('result', []);
        $ids = array_column($results, 'id');
        $scoreMap = [];

        foreach ($results as $r) {
            $scoreMap[$r['id']] = $r['score'];
        }

        if (empty($ids)) {
            return ['memories' => [], 'scores' => []];
        }

        $memories = BrainMemory::whereIn('id', $ids)
            ->forWorkspace($workspaceId)
            ->active()
            ->latestVersions()
            ->get()
            ->sortBy(fn (BrainMemory $m) => array_search($m->id, $ids))
            ->values();

        return [
            'memories' => $memories->map(fn (BrainMemory $m) => $m->toMcpContext(
                (float) ($scoreMap[$m->id] ?? 0.0)
            ))->all(),
        ];
    }

    /**
     * Remove a memory from both Qdrant and MariaDB.
     */
    public function forget(string $id): void
    {
        DB::connection('brain')->transaction(function () use ($id) {
            BrainMemory::where('id', $id)->delete();
            $this->qdrantDelete([$id]);
        });
    }

    /**
     * Ensure the Qdrant collection exists, creating it if needed.
     */
    public function ensureCollection(): void
    {
        $response = $this->http(5)
            ->get("{$this->qdrantUrl}/collections/{$this->collection}");

        if ($response->status() === 404) {
            $createResponse = $this->http(10)
                ->put("{$this->qdrantUrl}/collections/{$this->collection}", [
                    'vectors' => [
                        'size' => self::VECTOR_DIMENSION,
                        'distance' => 'Cosine',
                    ],
                ]);

            if (! $createResponse->successful()) {
                throw new \RuntimeException("Qdrant collection creation failed: {$createResponse->status()}");
            }

            Log::info("OpenBrain: created Qdrant collection '{$this->collection}'");
        }
    }

    /**
     * Build a Qdrant point payload.
     *
     * @param  array<string, mixed>  $metadata
     * @return array{id: string, payload: array<string, mixed>}
     */
    public function buildQdrantPayload(string $id, array $metadata): array
    {
        return [
            'id' => $id,
            'payload' => $metadata,
        ];
    }

    /**
     * Build a Qdrant filter from criteria.
     *
     * @param  array<string, mixed>  $criteria
     * @return array{must: array}
     */
    public function buildQdrantFilter(array $criteria): array
    {
        $must = [];

        if (isset($criteria['workspace_id'])) {
            $must[] = ['key' => 'workspace_id', 'match' => ['value' => $criteria['workspace_id']]];
        }

        if (isset($criteria['org'])) {
            $must[] = ['key' => 'org', 'match' => ['value' => $criteria['org']]];
        }

        if (isset($criteria['project'])) {
            $must[] = ['key' => 'project', 'match' => ['value' => $criteria['project']]];
        }

        if (isset($criteria['type'])) {
            if (is_array($criteria['type'])) {
                $must[] = ['key' => 'type', 'match' => ['any' => $criteria['type']]];
            } else {
                $must[] = ['key' => 'type', 'match' => ['value' => $criteria['type']]];
            }
        }

        if (isset($criteria['agent_id'])) {
            $must[] = ['key' => 'agent_id', 'match' => ['value' => $criteria['agent_id']]];
        }

        if (isset($criteria['min_confidence'])) {
            $must[] = ['key' => 'confidence', 'range' => ['gte' => $criteria['min_confidence']]];
        }

        return ['must' => $must];
    }

    /**
     * Upsert points into Qdrant.
     *
     * @param  array<array>  $points
     *
     * @throws \RuntimeException
     */
    private function qdrantUpsert(array $points): void
    {
        $response = $this->http(10)
            ->put("{$this->qdrantUrl}/collections/{$this->collection}/points", [
                'points' => $points,
            ]);

        if (! $response->successful()) {
            Log::error("Qdrant upsert failed: {$response->status()}", ['body' => $response->body()]);
            throw new \RuntimeException("Qdrant upsert failed: {$response->status()}");
        }
    }

    /**
     * Delete points from Qdrant by ID.
     *
     * @param  array<string>  $ids
     */
    private function qdrantDelete(array $ids): void
    {
        $response = $this->http(10)
            ->post("{$this->qdrantUrl}/collections/{$this->collection}/points/delete", [
                'points' => $ids,
            ]);

        if (! $response->successful()) {
            Log::warning("Qdrant delete failed: {$response->status()}", ['ids' => $ids, 'body' => $response->body()]);
        }
    }
}
