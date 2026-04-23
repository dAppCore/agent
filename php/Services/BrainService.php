<?php

// SPDX-License-Identifier: EUPL-1.2

declare(strict_types=1);

namespace Core\Mod\Agentic\Services;

use Core\Mod\Agentic\Models\BrainMemory;
use Illuminate\Http\Client\PendingRequest;
use Illuminate\Support\Facades\DB;
use Illuminate\Support\Facades\Http;
use Illuminate\Support\Facades\Log;

class BrainService
{
    private const DEFAULT_MODEL = 'embeddinggemma';

    private const VECTOR_DIMENSION = 768;

    private const ELASTIC_INDEX = 'brain_memories';

    public function __construct(
        private string $ollamaUrl = 'http://localhost:11434',
        private string $qdrantUrl = 'http://localhost:6334',
        private string $collection = 'openbrain',
        private string $embeddingModel = self::DEFAULT_MODEL,
        private bool $verifySsl = true,
        private string $elasticsearchUrl = 'http://127.0.0.1:9200',
    ) {}

    /**
     * Create an HTTP client with common settings.
     */
    private function http(int $timeout = 10): PendingRequest
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
     * Store a memory and queue asynchronous indexing.
     *
     * Creates the brain database record within a DB transaction and dispatches
     * EmbedMemory after commit so embedding, Qdrant, and Elasticsearch
     * indexing happen on the queue.
     *
     * @param  array<string, mixed>  $attributes  Fillable attributes for BrainMemory
     * @return BrainMemory The created memory
     */
    public function remember(array $attributes): BrainMemory
    {
        $attributes['indexed_at'] = null;

        $memory = DB::connection('brain')->transaction(function () use ($attributes) {
            $memory = BrainMemory::create($attributes);
            if ($memory->supersedes_id) {
                BrainMemory::where('id', $memory->supersedes_id)->delete();
            }

            return $memory;
        });

        \Core\Mod\Agentic\Jobs\EmbedMemory::dispatch($memory->id);

        return $memory;
    }

    /**
     * Semantic search: find memories similar to the query.
     *
     * @param  array<string, mixed>  $filter  Optional filter criteria
     * @return array{memories: array, scores: array<string, float>}
     */
    public function recall(
        string $query,
        int $topK,
        array $filter,
        int $workspaceId,
        array $keywords = [],
        array $boostKeywords = [],
    ): array {
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
        $scoreMap = $this->scoreQdrantResults(is_array($results) ? $results : []);
        $keywords = $this->normaliseKeywords($keywords);

        if ($keywords !== []) {
            $keywordScoreMap = $this->scoreElasticResults(
                $this->elasticSearch(implode(' ', $keywords), $filter, $topK),
            );

            foreach ($keywordScoreMap as $id => $score) {
                $scoreMap[$id] = max($scoreMap[$id] ?? 0.0, $score);
            }
        }

        if ($scoreMap === []) {
            return ['memories' => [], 'scores' => []];
        }

        $boostKeywords = $this->normaliseKeywords($boostKeywords);
        $boostMultiplier = $boostKeywords !== [] ? $this->boostKeywordMultiplier() : 1.0;
        $ranked = [];

        $memories = BrainMemory::whereIn('id', array_keys($scoreMap))
            ->forWorkspace($workspaceId)
            ->active()
            ->latestVersions()
            ->get();

        foreach ($memories as $memory) {
            $score = (float) ($scoreMap[$memory->id] ?? 0.0);

            if ($boostKeywords !== [] && $this->memoryContainsKeyword($memory, $boostKeywords)) {
                $score *= $boostMultiplier;
            }

            $ranked[] = [
                'memory' => $memory,
                'score' => $score,
            ];
        }

        usort($ranked, static fn (array $left, array $right): int => $right['score'] <=> $left['score']);
        $ranked = array_slice($ranked, 0, $topK);
        $finalScoreMap = [];

        return [
            'memories' => array_map(static function (array $item) use (&$finalScoreMap): array {
                /** @var BrainMemory $memory */
                $memory = $item['memory'];
                $score = (float) $item['score'];
                $finalScoreMap[$memory->id] = $score;

                return $memory->toMcpContext($score);
            }, $ranked),
            'scores' => $finalScoreMap,
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
     * Index a memory in Elasticsearch.
     */
    public function elasticIndex(BrainMemory $memory): void
    {
        $response = $this->http(10)
            ->put($this->elasticDocumentUrl($memory->id), $this->buildElasticDocument($memory));

        if (! $response->successful()) {
            Log::error("Elasticsearch index failed: {$response->status()}", ['id' => $memory->id, 'body' => $response->body()]);
            throw new \RuntimeException("Elasticsearch index failed: {$response->status()}");
        }
    }

    /**
     * Delete a memory from Elasticsearch.
     */
    public function elasticDelete(string $id): void
    {
        $response = $this->http(10)
            ->delete($this->elasticDocumentUrl($id));

        if (! $response->successful()) {
            Log::error("Elasticsearch delete failed: {$response->status()}", ['id' => $id, 'body' => $response->body()]);
            throw new \RuntimeException("Elasticsearch delete failed: {$response->status()}");
        }
    }

    /**
     * Search memories in Elasticsearch.
     *
     * @param  array<string, mixed>  $filters
     * @return array<string, mixed>
     */
    public function elasticSearch(string $query, array $filters = [], ?int $limit = null): array
    {
        $body = [
            'query' => [
                'bool' => [
                    'must' => [$this->buildElasticQuery($query)],
                    'filter' => $this->buildElasticFilters($filters),
                ],
            ],
        ];

        if ($limit !== null && $limit > 0) {
            $body['size'] = $limit;
        }

        $response = $this->http(10)
            ->post($this->elasticSearchUrl(), $body);

        if (! $response->successful()) {
            Log::error("Elasticsearch search failed: {$response->status()}", ['query' => $query, 'filters' => $filters, 'body' => $response->body()]);
            throw new \RuntimeException("Elasticsearch search failed: {$response->status()}");
        }

        $result = $response->json();

        return is_array($result) ? $result : [];
    }

    /**
     * @param  array<int, array<string, mixed>>  $results
     * @return array<string, float>
     */
    private function scoreQdrantResults(array $results): array
    {
        $scores = [];

        foreach ($results as $result) {
            $id = (string) ($result['id'] ?? '');
            if ($id === '') {
                continue;
            }

            $scores[$id] = (float) ($result['score'] ?? 0.0);
        }

        return $scores;
    }

    /**
     * @return array<string, float>
     */
    private function scoreElasticResults(array $result): array
    {
        $hits = $result['hits']['hits'] ?? [];
        if (! is_array($hits) || $hits === []) {
            return [];
        }

        $scores = [];
        foreach ($hits as $hit) {
            if (! is_array($hit)) {
                continue;
            }

            $id = (string) ($hit['_id'] ?? '');
            if ($id === '' && isset($hit['_source']) && is_array($hit['_source'])) {
                $id = (string) ($hit['_source']['id'] ?? '');
            }

            if ($id === '') {
                continue;
            }

            $scores[$id] = (float) ($hit['_score'] ?? 0.0);
        }

        return $scores;
    }

    /**
     * @param  array<int, mixed>  $keywords
     * @return array<int, string>
     */
    private function normaliseKeywords(array $keywords): array
    {
        return array_values(array_filter(array_map(
            static fn (mixed $keyword): string => is_string($keyword) ? trim($keyword) : '',
            $keywords,
        ), static fn (string $keyword): bool => $keyword !== ''));
    }

    private function boostKeywordMultiplier(): float
    {
        $configured = function_exists('config')
            ? config('mcp.brain.boost_keywords_multiplier', config('mcp.brain.keyword_boost', 1.5))
            : 1.5;
        $multiplier = is_numeric($configured) ? (float) $configured : 1.5;

        return $multiplier > 0.0 ? $multiplier : 1.5;
    }

    /**
     * @param  array<int, string>  $keywords
     */
    private function memoryContainsKeyword(BrainMemory $memory, array $keywords): bool
    {
        $haystack = mb_strtolower(implode(' ', array_filter([
            $memory->content,
            $memory->type,
            $memory->project,
            $memory->source,
            $memory->getAttribute('org'),
            implode(' ', $memory->tags ?? []),
        ], static fn (mixed $value): bool => is_string($value) && $value !== '')));

        foreach ($keywords as $keyword) {
            if (str_contains($haystack, mb_strtolower($keyword))) {
                return true;
            }
        }

        return false;
    }

    /**
     * Run an Elasticsearch aggregation query against brain memories.
     *
     * @param  array<string, mixed>  $body
     * @return array<string, mixed>
     */
    public function elasticAggregate(array $body): array
    {
        $response = $this->http(10)
            ->post($this->elasticSearchUrl(), $body);

        if (! $response->successful()) {
            Log::error("Elasticsearch aggregation failed: {$response->status()}", ['request' => $body, 'body' => $response->body()]);
            throw new \RuntimeException("Elasticsearch aggregation failed: {$response->status()}");
        }

        $result = $response->json();

        return is_array($result) ? $result : [];
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
     * Build an Elasticsearch document body from a memory.
     *
     * @return array<string, mixed>
     */
    private function buildElasticDocument(BrainMemory $memory): array
    {
        return [
            'id' => $memory->id,
            'content' => $memory->content,
            'type' => $memory->type,
            'tags' => $memory->tags ?? [],
            'project' => $memory->project,
            'workspace_id' => $memory->workspace_id,
            'org' => $memory->getAttribute('org'),
            'confidence' => $memory->confidence,
            'indexed_at' => $memory->indexed_at?->toIso8601String(),
        ];
    }

    /**
     * @return array<string, mixed>
     */
    private function buildElasticQuery(string $query): array
    {
        if ($query === '') {
            return ['match_all' => (object) []];
        }

        return [
            'multi_match' => [
                'query' => $query,
                'fields' => [
                    'content^3',
                    'type',
                    'tags',
                    'project',
                    'org',
                ],
            ],
        ];
    }

    /**
     * @param  array<string, mixed>  $filters
     * @return array<int, array<string, mixed>>
     */
    private function buildElasticFilters(array $filters): array
    {
        $clauses = [];

        foreach (['workspace_id', 'org', 'project', 'type'] as $field) {
            if (! isset($filters[$field])) {
                continue;
            }

            $clauses[] = is_array($filters[$field])
                ? ['terms' => [$field => $filters[$field]]]
                : ['term' => [$field => $filters[$field]]];
        }

        if (isset($filters['tags'])) {
            $clauses[] = is_array($filters['tags'])
                ? ['terms' => ['tags' => $filters['tags']]]
                : ['term' => ['tags' => $filters['tags']]];
        }

        if (isset($filters['min_confidence'])) {
            $clauses[] = ['range' => ['confidence' => ['gte' => $filters['min_confidence']]]];
        }

        return $clauses;
    }

    private function elasticDocumentUrl(string $id): string
    {
        return $this->elasticIndexUrl().'/_doc/'.rawurlencode($id);
    }

    private function elasticSearchUrl(): string
    {
        return $this->elasticIndexUrl().'/_search';
    }

    private function elasticIndexUrl(): string
    {
        return rtrim($this->elasticsearchUrl, '/').'/'.self::ELASTIC_INDEX;
    }

    /**
     * Upsert points into Qdrant.
     *
     * @param  array<array>  $points
     *
     * @throws \RuntimeException
     */
    public function qdrantUpsert(array $points): void
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
     *
     * @throws \RuntimeException
     */
    public function qdrantDelete(array $ids): void
    {
        $response = $this->http(10)
            ->post("{$this->qdrantUrl}/collections/{$this->collection}/points/delete", [
                'points' => $ids,
            ]);

        if (! $response->successful()) {
            Log::error("Qdrant delete failed: {$response->status()}", ['ids' => $ids, 'body' => $response->body()]);
            throw new \RuntimeException("Qdrant delete failed: {$response->status()}");
        }
    }
}
