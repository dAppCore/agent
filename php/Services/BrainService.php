<?php

// SPDX-License-Identifier: EUPL-1.2

declare(strict_types=1);

namespace Core\Mod\Agentic\Services;

use Core\Mod\Agentic\Jobs\DeleteFromIndex;
use Core\Mod\Agentic\Jobs\EmbedMemory;
use Core\Mod\Agentic\Models\BrainMemory;
use Illuminate\Auth\Access\AuthorizationException;
use Illuminate\Contracts\Cache\LockTimeoutException;
use Illuminate\Database\Eloquent\Builder;
use Illuminate\Http\Client\ConnectionException;
use Illuminate\Http\Client\PendingRequest;
use Illuminate\Http\Client\Response;
use Illuminate\Http\Request;
use Illuminate\Support\Facades\Cache;
use Illuminate\Support\Facades\DB;
use Illuminate\Support\Facades\Http;
use Illuminate\Support\Facades\Log;

class BrainService
{
    private const DEFAULT_MODEL = 'embeddinggemma';

    private const VECTOR_DIMENSION = 768;

    private const ELASTIC_INDEX = 'brain_memories';

    private const MAX_RETRY_DELAY_MS = 30000;

    private const MAX_HTTP_ATTEMPTS = 6;

    private const MEMORY_LOCK_TTL_SECONDS = 10;

    private const MEMORY_LOCK_WAIT_SECONDS = 5;

    private const MAX_SUPERSEDE_CHAIN_DEPTH = 100;

    private const MAX_CONTENT_BYTES = 65536;

    private const MAX_TAG_COUNT = 100;

    private const MAX_TAG_LENGTH = 128;

    private const MAX_ID_LENGTH = 64;

    private const MAX_AGENT_ID_LENGTH = 64;

    private const MAX_TYPE_LENGTH = 64;

    private const MAX_PROJECT_LENGTH = 128;

    private const MAX_ORG_LENGTH = 128;

    private const MAX_SEARCH_QUERY_BYTES = 2000;

    private const MAX_DISCOVERY_LIMIT = 100;

    private string $qdrantApiKey;

    public function __construct(
        private string $ollamaUrl = 'http://localhost:11434',
        private string $qdrantUrl = 'http://localhost:6334',
        private string $collection = 'openbrain',
        private string $embeddingModel = self::DEFAULT_MODEL,
        private bool $verifySsl = true,
        private string $elasticsearchUrl = 'http://127.0.0.1:9200',
        ?string $qdrantApiKey = null,
    ) {
        if ($qdrantApiKey === null && function_exists('config')) {
            $configuredQdrantApiKey = config('mcp.brain.qdrant.api_key', config('brain.qdrant.api_key', ''));
            $qdrantApiKey = is_string($configuredQdrantApiKey) ? $configuredQdrantApiKey : '';
        }

        $this->qdrantApiKey = trim((string) $qdrantApiKey);
    }

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
     * Create an HTTP client for Qdrant requests.
     */
    private function qdrantHttp(int $timeout = 10): PendingRequest
    {
        $request = $this->http($timeout);

        if ($this->qdrantApiKey === '') {
            return $request;
        }

        return $request->withHeaders([
            'api-key' => $this->qdrantApiKey,
        ]);
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
        $this->validateRememberInput($attributes);
        $this->assertAuthorisedOrgScope($attributes['org'] ?? null);

        $attributes['indexed_at'] = null;
        $cleanupIds = [];
        $requestedSupersededId = $this->normaliseMemoryId($attributes['supersedes_id'] ?? null);
        $remember = function (array $rememberAttributes) use (&$cleanupIds): BrainMemory {
            return DB::connection('brain')->transaction(function () use ($rememberAttributes, &$cleanupIds): BrainMemory {
                $memory = new BrainMemory;
                $memory->fill($rememberAttributes);
                $memory->save();

                $this->deleteSupersededMemory($memory->supersedes_id, $cleanupIds);

                return $memory;
            });
        };

        $memory = $requestedSupersededId !== null
            ? $this->withMemoryIndexLock($requestedSupersededId, function () use ($requestedSupersededId, $attributes, $remember): BrainMemory {
                $resolvedSupersededId = $this->resolveSupersedeHeadId($requestedSupersededId);
                $rememberAttributes = $attributes;
                // Retry calls should follow the current head rather than branch from a deleted ancestor.
                $rememberAttributes['supersedes_id'] = $resolvedSupersededId;

                if ($resolvedSupersededId === $requestedSupersededId) {
                    return $remember($rememberAttributes);
                }

                return $this->withMemoryIndexLock($resolvedSupersededId, fn (): BrainMemory => $remember($rememberAttributes));
            })
            : $remember($attributes);

        foreach ($cleanupIds as $cleanupId) {
            DeleteFromIndex::dispatch($cleanupId);
        }

        EmbedMemory::dispatch($memory->id);

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
        $this->validateMemoryFilters($filter);
        $this->assertAuthorisedOrgScope($filter['org'] ?? null);

        $vector = $this->embed($query);

        $filter['workspace_id'] = $workspaceId;
        $qdrantFilter = $this->buildQdrantFilter($filter);

        $response = $this->retryableHttp(10, fn (PendingRequest $request): Response => $request->post(
            "{$this->qdrantUrl}/collections/{$this->collection}/points/search",
            [
                'vector' => $vector,
                'filter' => $qdrantFilter,
                'limit' => $topK,
                'with_payload' => false,
            ],
        ));

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
     * Full-text discovery search with MariaDB fallback.
     *
     * @param  array<string, mixed>  $filters
     * @return array<int, array<string, mixed>>
     */
    public function search(string $query, int $workspaceId, array $filters = [], int $limit = 20): array
    {
        $this->validateSearchQuery($query);
        $this->validateDiscoveryLimit($limit);
        $this->validateMemoryFilters($filters);
        $this->assertAuthorisedOrgScope($filters['org'] ?? null);

        try {
            return $this->hydrateElasticSearchResults(
                $this->elasticSearch($query, array_merge($filters, ['workspace_id' => $workspaceId]), $limit),
                $workspaceId,
                $filters,
            );
        } catch (\RuntimeException $exception) {
            Log::warning('OpenBrain Elasticsearch search failed, falling back to MariaDB', [
                'message' => $exception->getMessage(),
                'workspace_id' => $workspaceId,
            ]);
        }

        return $this->brainQuery($workspaceId, $filters)
            ->where('content', 'like', '%'.$query.'%')
            ->orderByDesc('updated_at')
            ->limit($limit)
            ->get()
            ->map(static fn (BrainMemory $memory): array => $memory->toMcpContext())
            ->all();
    }

    /**
     * Discover the most common tags for a workspace scope.
     *
     * @return array<int, array{name: string, count: int}>
     */
    public function discoverTags(
        int $workspaceId,
        ?string $org = null,
        ?string $project = null,
        int $limit = 20,
    ): array {
        $this->validateDiscoveryLimit($limit);
        $this->validateMemoryFilters([
            'org' => $org,
            'project' => $project,
        ]);
        $this->assertAuthorisedOrgScope($org);

        $counts = [];

        $this->brainQuery($workspaceId, array_filter([
            'org' => $org,
            'project' => $project,
        ], static fn (mixed $value): bool => $value !== null && $value !== ''))
            ->select('tags')
            ->cursor()
            ->each(static function (BrainMemory $memory) use (&$counts): void {
                foreach ($memory->tags ?? [] as $tag) {
                    if (! is_string($tag)) {
                        continue;
                    }

                    $tag = trim($tag);

                    if ($tag === '') {
                        continue;
                    }

                    $counts[$tag] = ($counts[$tag] ?? 0) + 1;
                }
            });

        arsort($counts);

        $topCounts = array_slice($counts, 0, $limit, true);

        return array_values(array_map(
            static fn (string $tag, int $count): array => ['name' => $tag, 'count' => $count],
            array_keys($topCounts),
            array_values($topCounts),
        ));
    }

    /**
     * List org/project scopes with memory counts.
     *
     * @return array<int, array{org: ?string, count: int, projects: array<int, array{name: string, count: int}>}>
     */
    public function listScopes(int $workspaceId): array
    {
        $rows = $this->brainQuery($workspaceId)
            ->selectRaw("coalesce(org, '') as org_key, coalesce(project, '') as project_key, count(*) as cnt")
            ->groupBy('org_key', 'project_key')
            ->get();

        $scopes = [];

        foreach ($rows as $row) {
            $orgKey = is_string($row->org_key ?? null) ? $row->org_key : '';
            $projectKey = is_string($row->project_key ?? null) ? $row->project_key : '';

            if (! isset($scopes[$orgKey])) {
                $scopes[$orgKey] = [
                    'org' => $orgKey !== '' ? $orgKey : null,
                    'count' => 0,
                    'projects' => [],
                ];
            }

            $scopes[$orgKey]['count'] += (int) ($row->cnt ?? 0);

            if ($projectKey === '') {
                continue;
            }

            $scopes[$orgKey]['projects'][] = [
                'name' => $projectKey,
                'count' => (int) ($row->cnt ?? 0),
            ];
        }

        $scopes = array_values($scopes);

        foreach ($scopes as &$scope) {
            usort(
                $scope['projects'],
                static fn (array $left, array $right): int => $left['name'] <=> $right['name'],
            );
        }
        unset($scope);

        usort(
            $scopes,
            static fn (array $left, array $right): int => ($left['org'] ?? '') <=> ($right['org'] ?? ''),
        );

        return $scopes;
    }

    /**
     * Remove a memory from both Qdrant and MariaDB.
     */
    public function forget(string $id): void
    {
        $this->validateForgetInput($id);

        $memoryOrg = BrainMemory::query()
            ->whereKey($id)
            ->value('org');

        $this->assertAuthorisedOrgScope($memoryOrg);

        $deleted = $this->withMemoryIndexLock($id, fn (): bool => $this->deleteMemoryRecord($id));

        if ($deleted) {
            DeleteFromIndex::dispatch($id);
        }
    }

    /**
     * Ensure the Qdrant collection exists, creating it if needed.
     */
    public function ensureCollection(): void
    {
        $response = $this->retryableHttp(
            5,
            fn (PendingRequest $request): Response => $request->get("{$this->qdrantUrl}/collections/{$this->collection}")
        );

        if ($response->status() === 404) {
            $createResponse = $this->retryableHttp(10, fn (PendingRequest $request): Response => $request->put(
                "{$this->qdrantUrl}/collections/{$this->collection}",
                [
                    'vectors' => [
                        'size' => self::VECTOR_DIMENSION,
                        'distance' => 'Cosine',
                    ],
                ],
            ));

            if (! $createResponse->successful()) {
                throw new \RuntimeException("Qdrant collection creation failed: {$createResponse->status()}");
            }

            Log::info("OpenBrain: created Qdrant collection '{$this->collection}'");

            return;
        }

        if (! $response->successful()) {
            throw new \RuntimeException("Qdrant collection check failed: {$response->status()}");
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
        $this->withMemoryIndexLock($memory->id, function () use ($memory): void {
            $freshMemory = BrainMemory::query()->find($memory->id);

            if (! $freshMemory instanceof BrainMemory) {
                return;
            }

            $response = $this->http(10)
                ->put($this->elasticDocumentUrl($freshMemory->id), $this->buildElasticDocument($freshMemory));

            if (! $response->successful()) {
                Log::error("Elasticsearch index failed: {$response->status()}", ['id' => $freshMemory->id, 'body' => $response->body()]);
                throw new \RuntimeException("Elasticsearch index failed: {$response->status()}");
            }
        });
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
        $this->validateMemoryFilters($filters);
        $this->assertAuthorisedOrgScope($filters['org'] ?? null);

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
     * @param  array<string, mixed>  $attributes
     */
    private function validateRememberInput(array $attributes): void
    {
        $this->validateContent($attributes['content'] ?? null);
        $this->validateTags($attributes['tags'] ?? null);
        $this->validateStringMaxLength($attributes['supersedes_id'] ?? null, 'supersedes_id', self::MAX_ID_LENGTH);
        $this->validateStringMaxLength($attributes['agent_id'] ?? null, 'agent_id', self::MAX_AGENT_ID_LENGTH);
        $this->validateStringMaxLength($attributes['type'] ?? null, 'type', self::MAX_TYPE_LENGTH);
        $this->validateConfidence($attributes['confidence'] ?? null, 'confidence');
        $this->validateStringMaxLength($attributes['project'] ?? null, 'project', self::MAX_PROJECT_LENGTH);
        $this->validateStringMaxLength($attributes['org'] ?? null, 'org', self::MAX_ORG_LENGTH);
    }

    /**
     * @param  array<string, mixed>  $filters
     */
    private function validateMemoryFilters(array $filters): void
    {
        $this->validateStringMaxLength($filters['org'] ?? null, 'org', self::MAX_ORG_LENGTH);
        $this->validateStringMaxLength($filters['project'] ?? null, 'project', self::MAX_PROJECT_LENGTH);
        $this->validateStringOrArrayMaxLength($filters['type'] ?? null, 'type', self::MAX_TYPE_LENGTH);
        $this->validateStringMaxLength($filters['agent_id'] ?? null, 'agent_id', self::MAX_AGENT_ID_LENGTH);
        $this->validateTags($filters['tags'] ?? null, allowSingleTag: true);
        $this->validateConfidence($filters['min_confidence'] ?? null, 'min_confidence');
    }

    private function validateForgetInput(string $id): void
    {
        $this->validateStringMaxLength($id, 'id', self::MAX_ID_LENGTH);
    }

    private function validateSearchQuery(string $query): void
    {
        if (trim($query) === '') {
            throw new \InvalidArgumentException('query must not be empty');
        }

        $this->validateStringMaxLength($query, 'query', self::MAX_SEARCH_QUERY_BYTES);
    }

    private function validateDiscoveryLimit(int $limit): void
    {
        if ($limit < 1 || $limit > self::MAX_DISCOVERY_LIMIT) {
            throw new \InvalidArgumentException(
                sprintf('limit must be between 1 and %d', self::MAX_DISCOVERY_LIMIT)
            );
        }
    }

    private function validateContent(mixed $content): void
    {
        if ($content === null) {
            return;
        }

        if (! is_string($content)) {
            throw new \InvalidArgumentException('content must be a string');
        }

        if (strlen($content) > self::MAX_CONTENT_BYTES) {
            throw new \InvalidArgumentException(
                sprintf('content exceeds maximum length of %d bytes', self::MAX_CONTENT_BYTES)
            );
        }
    }

    private function validateTags(mixed $tags, bool $allowSingleTag = false): void
    {
        if ($tags === null) {
            return;
        }

        if ($allowSingleTag && is_string($tags)) {
            if (strlen($tags) > self::MAX_TAG_LENGTH) {
                throw new \InvalidArgumentException(
                    sprintf('tag exceeds maximum length of %d', self::MAX_TAG_LENGTH)
                );
            }

            return;
        }

        if (! is_array($tags)) {
            throw new \InvalidArgumentException('tags must be an array');
        }

        if (count($tags) > self::MAX_TAG_COUNT) {
            throw new \InvalidArgumentException(
                sprintf('tags array exceeds maximum size of %d', self::MAX_TAG_COUNT)
            );
        }

        foreach ($tags as $index => $tag) {
            if (! is_string($tag)) {
                throw new \InvalidArgumentException(
                    sprintf('tag at index %s must be a string', (string) $index)
                );
            }

            if (strlen($tag) > self::MAX_TAG_LENGTH) {
                throw new \InvalidArgumentException(
                    sprintf('tag at index %s exceeds maximum length of %d', (string) $index, self::MAX_TAG_LENGTH)
                );
            }
        }
    }

    private function validateStringMaxLength(mixed $value, string $field, int $maxLength): void
    {
        if ($value === null) {
            return;
        }

        if (! is_string($value)) {
            throw new \InvalidArgumentException(sprintf('%s must be a string', $field));
        }

        if (strlen($value) > $maxLength) {
            throw new \InvalidArgumentException(
                sprintf('%s exceeds maximum length of %d', $field, $maxLength)
            );
        }
    }

    private function validateStringOrArrayMaxLength(mixed $value, string $field, int $maxLength): void
    {
        if ($value === null) {
            return;
        }

        if (is_array($value)) {
            foreach ($value as $index => $item) {
                if (! is_string($item)) {
                    throw new \InvalidArgumentException(
                        sprintf('%s at index %s must be a string', $field, (string) $index)
                    );
                }

                if (strlen($item) > $maxLength) {
                    throw new \InvalidArgumentException(
                        sprintf('%s at index %s exceeds maximum length of %d', $field, (string) $index, $maxLength)
                    );
                }
            }

            return;
        }

        $this->validateStringMaxLength($value, $field, $maxLength);
    }

    private function validateConfidence(mixed $value, string $field): void
    {
        if ($value === null) {
            return;
        }

        if (! is_numeric($value)) {
            throw new \InvalidArgumentException(sprintf('%s must be between 0.0 and 1.0', $field));
        }

        $confidence = (float) $value;

        if ($confidence < 0.0 || $confidence > 1.0) {
            throw new \InvalidArgumentException(sprintf('%s must be between 0.0 and 1.0', $field));
        }
    }

    /**
     * @throws AuthorizationException
     */
    private function assertAuthorisedOrgScope(mixed $requestedOrg): void
    {
        $authorisedOrgs = $this->authorisedOrgScopes();
        if ($authorisedOrgs === []) {
            return;
        }

        foreach ($this->normaliseOrgScopes($requestedOrg) as $org) {
            if (! in_array($org, $authorisedOrgs, true)) {
                throw new AuthorizationException(
                    sprintf("Organisation scope '%s' is not authorised for this authenticated workspace.", $org)
                );
            }
        }
    }

    /**
     * @return array<int, string>
     */
    private function authorisedOrgScopes(): array
    {
        $request = $this->currentRequest();
        if (! $request instanceof Request) {
            return [];
        }

        $context = $request->attributes->get('mcp_workspace_context');
        if (is_array($context)) {
            $authorisedOrgs = $this->normaliseOrgScopes(
                $context['authorised_orgs'] ?? $context['authorized_orgs'] ?? null
            );

            if ($authorisedOrgs !== []) {
                return $authorisedOrgs;
            }

            $contextOrg = $this->resolveOrgScopeFromSource($context, [
                'org',
                'primary_org',
                'organisation',
                'organization',
            ]);

            if ($contextOrg !== null) {
                return [$contextOrg];
            }

            $workspaceOrg = $this->resolveOrgScopeFromSource($context['workspace'] ?? null, [
                'org',
                'organisation',
                'organization',
                'slug',
            ]);

            if ($workspaceOrg !== null) {
                return [$workspaceOrg];
            }
        }

        $workspace = $request->attributes->get('workspace') ?? $request->attributes->get('mcp_workspace');
        $workspaceOrg = $this->resolveOrgScopeFromSource($workspace, [
            'org',
            'organisation',
            'organization',
            'slug',
        ]);

        return $workspaceOrg !== null ? [$workspaceOrg] : [];
    }

    private function currentRequest(): ?Request
    {
        if (! function_exists('app') || ! app()->bound('request')) {
            return null;
        }

        $request = app('request');

        return $request instanceof Request ? $request : null;
    }

    private function applyAuthorisedOrgScopeQuery(Builder $query, mixed $requestedOrg = null): void
    {
        if ($requestedOrg !== null && $requestedOrg !== '') {
            return;
        }

        $authorisedOrgs = $this->authorisedOrgScopes();

        if ($authorisedOrgs === []) {
            return;
        }

        $query->where(function (Builder $scopeQuery) use ($authorisedOrgs): void {
            $scopeQuery->whereNull('org')
                ->orWhereIn('org', $authorisedOrgs);
        });
    }

    /**
     * @param  array<int, string>  $keys
     */
    private function resolveOrgScopeFromSource(mixed $source, array $keys): ?string
    {
        if (is_array($source)) {
            foreach ($keys as $key) {
                $scope = $this->normaliseSingleOrgScope($source[$key] ?? null);

                if ($scope !== null) {
                    return $scope;
                }
            }

            return null;
        }

        if (! is_object($source)) {
            return null;
        }

        foreach ($keys as $key) {
            $value = null;

            if (method_exists($source, 'getAttribute')) {
                $value = $source->getAttribute($key);
            }

            if ($value === null && isset($source->{$key})) {
                $value = $source->{$key};
            }

            $scope = $this->normaliseSingleOrgScope($value);
            if ($scope !== null) {
                return $scope;
            }
        }

        return null;
    }

    /**
     * @return array<int, string>
     */
    private function normaliseOrgScopes(mixed $value): array
    {
        if (is_string($value)) {
            $scope = $this->normaliseSingleOrgScope($value);

            return $scope !== null ? [$scope] : [];
        }

        if (! is_array($value)) {
            return [];
        }

        $scopes = [];

        foreach ($value as $item) {
            $scope = $this->normaliseSingleOrgScope($item);

            if ($scope !== null) {
                $scopes[] = $scope;
            }
        }

        return array_values(array_unique($scopes));
    }

    private function normaliseSingleOrgScope(mixed $value): ?string
    {
        if (! is_string($value)) {
            return null;
        }

        $scope = trim($value);

        return $scope !== '' ? $scope : null;
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
     * @param  array<string, mixed>  $filters
     */
    private function brainQuery(int $workspaceId, array $filters = []): Builder
    {
        $query = BrainMemory::query()
            ->forWorkspace($workspaceId)
            ->active()
            ->latestVersions();

        $this->applyAuthorisedOrgScopeQuery($query, $filters['org'] ?? null);

        if (isset($filters['org'])) {
            is_array($filters['org'])
                ? $query->whereIn('org', $filters['org'])
                : $query->where('org', $filters['org']);
        }

        if (isset($filters['project'])) {
            $query->where('project', $filters['project']);
        }

        if (isset($filters['type'])) {
            is_array($filters['type'])
                ? $query->whereIn('type', $filters['type'])
                : $query->where('type', $filters['type']);
        }

        if (isset($filters['agent_id'])) {
            $query->where('agent_id', $filters['agent_id']);
        }

        if (isset($filters['tags'])) {
            $tags = is_array($filters['tags']) ? $filters['tags'] : [$filters['tags']];

            $query->where(function (Builder $tagQuery) use ($tags): void {
                foreach ($tags as $tag) {
                    $tagQuery->orWhereJsonContains('tags', $tag);
                }
            });
        }

        if (isset($filters['min_confidence'])) {
            $query->where('confidence', '>=', (float) $filters['min_confidence']);
        }

        return $query;
    }

    /**
     * @param  array<string, mixed>  $result
     * @param  array<string, mixed>  $filters
     * @return array<int, array<string, mixed>>
     */
    private function hydrateElasticSearchResults(array $result, int $workspaceId, array $filters): array
    {
        $hits = $result['hits']['hits'] ?? [];

        if (! is_array($hits) || $hits === []) {
            return [];
        }

        $ids = [];
        $scores = [];

        foreach ($hits as $hit) {
            if (! is_array($hit)) {
                continue;
            }

            $id = $hit['_id'] ?? ($hit['_source']['id'] ?? null);

            if (! is_string($id) || $id === '' || in_array($id, $ids, true)) {
                continue;
            }

            $ids[] = $id;
            $scores[$id] = (float) ($hit['_score'] ?? 0.0);
        }

        if ($ids === []) {
            return [];
        }

        $memoryMap = $this->brainQuery($workspaceId, $filters)
            ->whereIn('id', $ids)
            ->get()
            ->keyBy('id');

        $memories = [];

        foreach ($ids as $id) {
            $memory = $memoryMap->get($id);

            if ($memory instanceof BrainMemory) {
                $memories[] = $memory->toMcpContext((float) ($scores[$id] ?? 0.0));
            }
        }

        return $memories;
    }

    /**
     * Build a Qdrant filter from criteria.
     *
     * @param  array<string, mixed>  $criteria
     * @return array{must: array}
     */
    public function buildQdrantFilter(array $criteria): array
    {
        $this->validateMemoryFilters($criteria);

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
        foreach ($points as $point) {
            $memoryId = $this->normaliseMemoryId($point['id'] ?? null);

            if ($memoryId === null) {
                continue;
            }

            $this->withMemoryIndexLock($memoryId, function () use ($memoryId, $point): void {
                if (! $this->memoryExists($memoryId)) {
                    return;
                }

                $response = $this->retryableHttp(10, fn (PendingRequest $request): Response => $request->put(
                    "{$this->qdrantUrl}/collections/{$this->collection}/points",
                    [
                        'points' => [$point],
                    ],
                ));

                if (! $response->successful()) {
                    Log::error("Qdrant upsert failed: {$response->status()}", ['body' => $response->body()]);
                    throw new \RuntimeException("Qdrant upsert failed: {$response->status()}");
                }
            });
        }
    }

    private function deleteMemoryRecord(string $id): bool
    {
        return DB::connection('brain')->transaction(function () use ($id): bool {
            $memory = BrainMemory::query()
                ->select(['id'])
                ->find($id);

            if (! $memory instanceof BrainMemory) {
                return false;
            }

            $memory->delete();

            return true;
        });
    }

    /**
     * @param  array<int, string>  $cleanupIds
     */
    private function deleteSupersededMemory(?string $supersededId, array &$cleanupIds): void
    {
        if ($supersededId === null || $supersededId === '') {
            return;
        }

        $superseded = BrainMemory::query()->find($supersededId);

        if (! $superseded instanceof BrainMemory) {
            throw new \RuntimeException("Superseded memory head vanished during delete: {$supersededId}");
        }

        $cleanupIds[] = $superseded->id;
        $superseded->delete();
    }

    private function resolveSupersedeHeadId(string $supersededId): string
    {
        $currentId = $supersededId;
        $visited = [];
        $depth = 0;

        while (true) {
            if (isset($visited[$currentId])) {
                throw new \RuntimeException("Detected cycle while resolving supersede chain: {$supersededId}");
            }

            if ($depth >= self::MAX_SUPERSEDE_CHAIN_DEPTH) {
                throw new \RuntimeException(
                    "Supersede chain exceeded ".self::MAX_SUPERSEDE_CHAIN_DEPTH." hops: {$supersededId}"
                );
            }

            $visited[$currentId] = true;

            $successor = BrainMemory::withTrashed()
                ->where('supersedes_id', $currentId)
                ->orderByDesc('created_at')
                ->orderByDesc('id')
                ->first();

            if ($successor instanceof BrainMemory) {
                $currentId = $successor->id;
                $depth++;

                continue;
            }

            $current = BrainMemory::withTrashed()->find($currentId);

            if (! $current instanceof BrainMemory) {
                throw new \InvalidArgumentException("Superseded memory not found: {$supersededId}");
            }

            if ($current->trashed()) {
                throw new \InvalidArgumentException("Superseded memory has no live head: {$supersededId}");
            }

            return $current->id;
        }
    }

    private function memoryExists(string $id): bool
    {
        return BrainMemory::query()->whereKey($id)->exists();
    }

    private function normaliseMemoryId(mixed $memoryId): ?string
    {
        return is_string($memoryId) && $memoryId !== ''
            ? $memoryId
            : null;
    }

    private function memoryLockKey(string $memoryId): string
    {
        return 'brain:memory:index:'.$memoryId;
    }

    private function withMemoryIndexLock(string $memoryId, callable $callback): mixed
    {
        $lock = Cache::lock($this->memoryLockKey($memoryId), self::MEMORY_LOCK_TTL_SECONDS);

        try {
            $lock->block(self::MEMORY_LOCK_WAIT_SECONDS);

            return $callback();
        } catch (LockTimeoutException $exception) {
            throw new \RuntimeException("Timed out acquiring memory index lock: {$memoryId}", 0, $exception);
        } finally {
            rescue(static fn (): mixed => $lock->release(), report: false);
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
        $response = $this->retryableHttp(10, fn (PendingRequest $request): Response => $request->post(
            "{$this->qdrantUrl}/collections/{$this->collection}/points/delete",
            [
                'points' => $ids,
            ],
        ));

        if (! $response->successful()) {
            Log::error("Qdrant delete failed: {$response->status()}", ['ids' => $ids, 'body' => $response->body()]);
            throw new \RuntimeException("Qdrant delete failed: {$response->status()}");
        }
    }

    /**
     * Retry transient Qdrant HTTP failures with a small exponential backoff.
     *
     * Retries 408, 429, and 503 responses plus connection failures. Other
     * responses are returned immediately so callers can fail fast.
     *
     * @param  callable(PendingRequest): Response  $buildRequest
     */
    private function retryableHttp(
        int $timeout,
        callable $buildRequest,
        int $maxAttempts = self::MAX_HTTP_ATTEMPTS,
    ): Response
    {
        $lastConnectionException = null;

        for ($attempt = 1; $attempt <= $maxAttempts; $attempt++) {
            try {
                $response = $buildRequest($this->qdrantHttp($timeout));
            } catch (ConnectionException $exception) {
                $lastConnectionException = $exception;

                if ($attempt === $maxAttempts) {
                    break;
                }

                $this->sleepMilliseconds($this->retryDelayMilliseconds(null, $attempt));

                continue;
            }

            if (! $this->shouldRetryResponse($response) || $attempt === $maxAttempts) {
                return $response;
            }

            $this->sleepMilliseconds($this->retryDelayMilliseconds($response, $attempt));
        }

        throw new \RuntimeException(
            sprintf(
                'Qdrant request failed after %d attempts: %s',
                $maxAttempts,
                $lastConnectionException?->getMessage() ?? 'connection error'
            ),
            0,
            $lastConnectionException,
        );
    }

    protected function sleepMilliseconds(int $milliseconds): void
    {
        usleep($milliseconds * 1000);
    }

    private function shouldRetryResponse(Response $response): bool
    {
        $status = $response->status();

        return $status === 408 || $status === 429 || $status === 503;
    }

    private function retryDelayMilliseconds(?Response $response, int $attempt): int
    {
        $retryAfter = $response?->header('Retry-After');
        if (is_string($retryAfter) && $retryAfter !== '') {
            $delay = $this->parseRetryAfterMilliseconds($retryAfter);

            if ($delay !== null) {
                return $delay;
            }
        }

        $delays = [100, 300, 900];

        return $delays[$attempt - 1] ?? 900;
    }

    private function parseRetryAfterMilliseconds(string $retryAfter): ?int
    {
        if (is_numeric($retryAfter)) {
            return min(max((int) $retryAfter, 0) * 1000, self::MAX_RETRY_DELAY_MS);
        }

        $retryAt = strtotime($retryAfter);
        if ($retryAt === false) {
            return null;
        }

        return min(max($retryAt - time(), 0) * 1000, self::MAX_RETRY_DELAY_MS);
    }
}
