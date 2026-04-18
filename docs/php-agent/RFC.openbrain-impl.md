# OpenBrain Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Shared vector-indexed knowledge store for all agents, accessible via 4 MCP tools (`brain_remember`, `brain_recall`, `brain_forget`, `brain_list`).

**Architecture:** MariaDB table in php-agentic for relational data. Qdrant collection for vector embeddings. Ollama for embedding generation. Go bridge in go-ai for CLI agents.

**Tech Stack:** PHP 8.4 / Laravel / Pest, Go 1.26, Qdrant REST API, Ollama embeddings API, MariaDB

**Prerequisites:**
- Qdrant container running on de1 (deploy via Ansible — separate task)
- Ollama with `nomic-embed-text` model pulled (`ollama pull nomic-embed-text`)

---

### Task 1: Migration + BrainMemory Model

**Files:**
- Create: `Migrations/0001_01_01_000004_create_brain_memories_table.php`
- Create: `Models/BrainMemory.php`

**Step 1: Write the migration**

```php
<?php

declare(strict_types=1);

use Illuminate\Database\Migrations\Migration;
use Illuminate\Database\Schema\Blueprint;
use Illuminate\Support\Facades\Schema;

return new class extends Migration
{
    public function up(): void
    {
        Schema::disableForeignKeyConstraints();

        if (! Schema::hasTable('brain_memories')) {
            Schema::create('brain_memories', function (Blueprint $table) {
                $table->uuid('id')->primary();
                $table->foreignId('workspace_id')->constrained()->cascadeOnDelete();
                $table->string('agent_id', 64);
                $table->string('type', 32)->index();
                $table->text('content');
                $table->json('tags')->nullable();
                $table->string('project', 128)->nullable()->index();
                $table->float('confidence')->default(1.0);
                $table->uuid('supersedes_id')->nullable();
                $table->timestamp('expires_at')->nullable();
                $table->timestamps();
                $table->softDeletes();

                $table->index('workspace_id');
                $table->index('agent_id');
                $table->index(['workspace_id', 'type']);
                $table->index(['workspace_id', 'project']);
                $table->foreign('supersedes_id')
                    ->references('id')
                    ->on('brain_memories')
                    ->nullOnDelete();
            });
        }

        Schema::enableForeignKeyConstraints();
    }

    public function down(): void
    {
        Schema::dropIfExists('brain_memories');
    }
};
```

**Step 2: Write the model**

```php
<?php

declare(strict_types=1);

namespace Core\Mod\Agentic\Models;

use Core\Tenant\Concerns\BelongsToWorkspace;
use Core\Tenant\Models\Workspace;
use Illuminate\Database\Eloquent\Builder;
use Illuminate\Database\Eloquent\Concerns\HasUuids;
use Illuminate\Database\Eloquent\Model;
use Illuminate\Database\Eloquent\Relations\BelongsTo;
use Illuminate\Database\Eloquent\Relations\HasMany;
use Illuminate\Database\Eloquent\SoftDeletes;

class BrainMemory extends Model
{
    use BelongsToWorkspace;
    use HasUuids;
    use SoftDeletes;

    public const TYPE_DECISION = 'decision';
    public const TYPE_OBSERVATION = 'observation';
    public const TYPE_CONVENTION = 'convention';
    public const TYPE_RESEARCH = 'research';
    public const TYPE_PLAN = 'plan';
    public const TYPE_BUG = 'bug';
    public const TYPE_ARCHITECTURE = 'architecture';

    public const VALID_TYPES = [
        self::TYPE_DECISION,
        self::TYPE_OBSERVATION,
        self::TYPE_CONVENTION,
        self::TYPE_RESEARCH,
        self::TYPE_PLAN,
        self::TYPE_BUG,
        self::TYPE_ARCHITECTURE,
    ];

    protected $table = 'brain_memories';

    protected $fillable = [
        'workspace_id',
        'agent_id',
        'type',
        'content',
        'tags',
        'project',
        'confidence',
        'supersedes_id',
        'expires_at',
    ];

    protected $casts = [
        'tags' => 'array',
        'confidence' => 'float',
        'expires_at' => 'datetime',
    ];

    public function workspace(): BelongsTo
    {
        return $this->belongsTo(Workspace::class);
    }

    public function supersedes(): BelongsTo
    {
        return $this->belongsTo(self::class, 'supersedes_id');
    }

    public function supersededBy(): HasMany
    {
        return $this->hasMany(self::class, 'supersedes_id');
    }

    public function scopeForWorkspace(Builder $query, int $workspaceId): Builder
    {
        return $query->where('workspace_id', $workspaceId);
    }

    public function scopeOfType(Builder $query, string|array $type): Builder
    {
        return is_array($type)
            ? $query->whereIn('type', $type)
            : $query->where('type', $type);
    }

    public function scopeForProject(Builder $query, ?string $project): Builder
    {
        return $project
            ? $query->where('project', $project)
            : $query;
    }

    public function scopeByAgent(Builder $query, ?string $agentId): Builder
    {
        return $agentId
            ? $query->where('agent_id', $agentId)
            : $query;
    }

    public function scopeActive(Builder $query): Builder
    {
        return $query->where(function (Builder $q) {
            $q->whereNull('expires_at')
              ->orWhere('expires_at', '>', now());
        });
    }

    public function scopeLatestVersions(Builder $query): Builder
    {
        return $query->whereDoesntHave('supersededBy', function (Builder $q) {
            $q->whereNull('deleted_at');
        });
    }

    public function getSupersessionDepth(): int
    {
        $count = 0;
        $current = $this;
        while ($current->supersedes_id) {
            $count++;
            $current = self::withTrashed()->find($current->supersedes_id);
            if (! $current) {
                break;
            }
        }

        return $count;
    }

    public function toMcpContext(): array
    {
        return [
            'id' => $this->id,
            'agent_id' => $this->agent_id,
            'type' => $this->type,
            'content' => $this->content,
            'tags' => $this->tags ?? [],
            'project' => $this->project,
            'confidence' => $this->confidence,
            'supersedes_id' => $this->supersedes_id,
            'supersedes_count' => $this->getSupersessionDepth(),
            'expires_at' => $this->expires_at?->toIso8601String(),
            'created_at' => $this->created_at?->toIso8601String(),
            'updated_at' => $this->updated_at?->toIso8601String(),
        ];
    }
}
```

**Step 3: Run migration locally to verify**

Run: `cd /Users/snider/Code/php-agentic && php artisan migrate --path=Migrations`
Expected: Migration runs without errors (or skip if no local DB — verify on deploy)

**Step 4: Commit**

```bash
cd /Users/snider/Code/php-agentic
git add Migrations/0001_01_01_000004_create_brain_memories_table.php Models/BrainMemory.php
git commit -m "feat(brain): add BrainMemory model and migration"
```

---

### Task 2: BrainService — Ollama embeddings + Qdrant client

**Files:**
- Create: `Services/BrainService.php`
- Create: `tests/Unit/BrainServiceTest.php`

**Step 1: Write the failing test**

```php
<?php

declare(strict_types=1);

use Core\Mod\Agentic\Services\BrainService;

it('builds qdrant upsert payload correctly', function () {
    $service = new BrainService(
        ollamaUrl: 'http://localhost:11434',
        qdrantUrl: 'http://localhost:6334',
        collection: 'openbrain_test',
    );

    $payload = $service->buildQdrantPayload('test-uuid', [
        'workspace_id' => 1,
        'agent_id' => 'virgil',
        'type' => 'decision',
        'tags' => ['scoring'],
        'project' => 'eaas',
        'confidence' => 0.9,
        'created_at' => '2026-03-03T00:00:00Z',
    ]);

    expect($payload)->toHaveKey('id', 'test-uuid');
    expect($payload)->toHaveKey('payload');
    expect($payload['payload']['agent_id'])->toBe('virgil');
    expect($payload['payload']['type'])->toBe('decision');
    expect($payload['payload']['tags'])->toBe(['scoring']);
});

it('builds qdrant search filter correctly', function () {
    $service = new BrainService(
        ollamaUrl: 'http://localhost:11434',
        qdrantUrl: 'http://localhost:6334',
        collection: 'openbrain_test',
    );

    $filter = $service->buildQdrantFilter([
        'workspace_id' => 1,
        'project' => 'eaas',
        'type' => ['decision', 'architecture'],
        'min_confidence' => 0.5,
    ]);

    expect($filter)->toHaveKey('must');
    expect($filter['must'])->toHaveCount(4);
});
```

**Step 2: Run test to verify it fails**

Run: `cd /Users/snider/Code/php-agentic && ./vendor/bin/pest tests/Unit/BrainServiceTest.php`
Expected: FAIL — class not found

**Step 3: Write the service**

```php
<?php

declare(strict_types=1);

namespace Core\Mod\Agentic\Services;

use Core\Mod\Agentic\Models\BrainMemory;
use Illuminate\Support\Facades\Http;
use Illuminate\Support\Facades\Log;

class BrainService
{
    private const EMBEDDING_MODEL = 'nomic-embed-text';
    private const VECTOR_DIMENSION = 768;

    public function __construct(
        private string $ollamaUrl = 'http://localhost:11434',
        private string $qdrantUrl = 'http://localhost:6334',
        private string $collection = 'openbrain',
    ) {}

    /**
     * Generate embedding vector for text content via Ollama.
     *
     * @return float[]
     */
    public function embed(string $text): array
    {
        $response = Http::timeout(30)
            ->post("{$this->ollamaUrl}/api/embeddings", [
                'model' => self::EMBEDDING_MODEL,
                'prompt' => $text,
            ]);

        if (! $response->successful()) {
            throw new \RuntimeException("Ollama embedding failed: {$response->status()}");
        }

        return $response->json('embedding');
    }

    /**
     * Store a memory: insert into MariaDB, embed, upsert into Qdrant.
     */
    public function remember(BrainMemory $memory): void
    {
        $vector = $this->embed($memory->content);

        $payload = $this->buildQdrantPayload($memory->id, [
            'workspace_id' => $memory->workspace_id,
            'agent_id' => $memory->agent_id,
            'type' => $memory->type,
            'tags' => $memory->tags ?? [],
            'project' => $memory->project,
            'confidence' => $memory->confidence,
            'created_at' => $memory->created_at->toIso8601String(),
        ]);
        $payload['vector'] = $vector;

        $this->qdrantUpsert([$payload]);

        // If superseding, remove old point from Qdrant
        if ($memory->supersedes_id) {
            $this->qdrantDelete([$memory->supersedes_id]);
            BrainMemory::where('id', $memory->supersedes_id)->delete();
        }
    }

    /**
     * Semantic search: embed query, search Qdrant, hydrate from MariaDB.
     *
     * @return array{memories: array, scores: array}
     */
    public function recall(string $query, int $topK, array $filter, int $workspaceId): array
    {
        $vector = $this->embed($query);

        $filter['workspace_id'] = $workspaceId;
        $qdrantFilter = $this->buildQdrantFilter($filter);

        $response = Http::timeout(10)
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
            ->active()
            ->latestVersions()
            ->get()
            ->sortBy(fn (BrainMemory $m) => array_search($m->id, $ids))
            ->values();

        return [
            'memories' => $memories->map(fn (BrainMemory $m) => $m->toMcpContext())->all(),
            'scores' => $scoreMap,
        ];
    }

    /**
     * Soft-delete a memory from MariaDB and remove from Qdrant.
     */
    public function forget(string $id): void
    {
        $this->qdrantDelete([$id]);
        BrainMemory::where('id', $id)->delete();
    }

    /**
     * Ensure the Qdrant collection exists, create if not.
     */
    public function ensureCollection(): void
    {
        $response = Http::timeout(5)
            ->get("{$this->qdrantUrl}/collections/{$this->collection}");

        if ($response->status() === 404) {
            Http::timeout(10)
                ->put("{$this->qdrantUrl}/collections/{$this->collection}", [
                    'vectors' => [
                        'size' => self::VECTOR_DIMENSION,
                        'distance' => 'Cosine',
                    ],
                ]);
            Log::info("OpenBrain: created Qdrant collection '{$this->collection}'");
        }
    }

    /**
     * Build a Qdrant point payload from memory metadata.
     */
    public function buildQdrantPayload(string $id, array $metadata): array
    {
        return [
            'id' => $id,
            'payload' => $metadata,
        ];
    }

    /**
     * Build a Qdrant filter from search criteria.
     */
    public function buildQdrantFilter(array $criteria): array
    {
        $must = [];

        if (isset($criteria['workspace_id'])) {
            $must[] = ['key' => 'workspace_id', 'match' => ['value' => $criteria['workspace_id']]];
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

    private function qdrantUpsert(array $points): void
    {
        $response = Http::timeout(10)
            ->put("{$this->qdrantUrl}/collections/{$this->collection}/points", [
                'points' => $points,
            ]);

        if (! $response->successful()) {
            Log::error("Qdrant upsert failed: {$response->status()}", ['body' => $response->body()]);
            throw new \RuntimeException("Qdrant upsert failed: {$response->status()}");
        }
    }

    private function qdrantDelete(array $ids): void
    {
        Http::timeout(10)
            ->post("{$this->qdrantUrl}/collections/{$this->collection}/points/delete", [
                'points' => $ids,
            ]);
    }
}
```

**Step 4: Run tests to verify they pass**

Run: `cd /Users/snider/Code/php-agentic && ./vendor/bin/pest tests/Unit/BrainServiceTest.php`
Expected: PASS (unit tests only test payload/filter building, no external services)

**Step 5: Commit**

```bash
cd /Users/snider/Code/php-agentic
git add Services/BrainService.php tests/Unit/BrainServiceTest.php
git commit -m "feat(brain): add BrainService with Ollama embeddings and Qdrant client"
```

---

### Task 3: BrainRemember MCP Tool

**Files:**
- Create: `Mcp/Tools/Agent/Brain/BrainRemember.php`
- Create: `tests/Unit/Tools/BrainRememberTest.php`

**Step 1: Write the failing test**

```php
<?php

declare(strict_types=1);

use Core\Mod\Agentic\Mcp\Tools\Agent\Brain\BrainRemember;

it('has correct name and category', function () {
    $tool = new BrainRemember();
    expect($tool->name())->toBe('brain_remember');
    expect($tool->category())->toBe('brain');
});

it('requires write scope', function () {
    $tool = new BrainRemember();
    expect($tool->requiredScopes())->toContain('write');
});

it('requires content in input schema', function () {
    $tool = new BrainRemember();
    $schema = $tool->inputSchema();
    expect($schema['required'])->toContain('content');
    expect($schema['required'])->toContain('type');
});

it('returns error when content is missing', function () {
    $tool = new BrainRemember();
    $result = $tool->handle([], ['workspace_id' => 1, 'agent_id' => 'virgil']);
    expect($result)->toHaveKey('error');
});

it('returns error when workspace_id is missing', function () {
    $tool = new BrainRemember();
    $result = $tool->handle([
        'content' => 'Test memory',
        'type' => 'observation',
    ], []);
    expect($result)->toHaveKey('error');
});
```

**Step 2: Run test to verify it fails**

Run: `cd /Users/snider/Code/php-agentic && ./vendor/bin/pest tests/Unit/Tools/BrainRememberTest.php`
Expected: FAIL — class not found

**Step 3: Write the tool**

```php
<?php

declare(strict_types=1);

namespace Core\Mod\Agentic\Mcp\Tools\Agent\Brain;

use Core\Mcp\Dependencies\ToolDependency;
use Core\Mod\Agentic\Mcp\Tools\Agent\AgentTool;
use Core\Mod\Agentic\Models\BrainMemory;
use Core\Mod\Agentic\Services\BrainService;

class BrainRemember extends AgentTool
{
    protected string $category = 'brain';
    protected array $scopes = ['write'];

    public function dependencies(): array
    {
        return [
            ToolDependency::contextExists('workspace_id', 'Workspace context required'),
        ];
    }

    public function name(): string
    {
        return 'brain_remember';
    }

    public function description(): string
    {
        return 'Store a memory in the shared agent knowledge graph. All agents can later recall this via semantic search.';
    }

    public function inputSchema(): array
    {
        return [
            'type' => 'object',
            'properties' => [
                'content' => [
                    'type' => 'string',
                    'description' => 'The knowledge to remember (markdown text)',
                ],
                'type' => [
                    'type' => 'string',
                    'enum' => BrainMemory::VALID_TYPES,
                    'description' => 'Category: decision, observation, convention, research, plan, bug, architecture',
                ],
                'tags' => [
                    'type' => 'array',
                    'items' => ['type' => 'string'],
                    'description' => 'Topic tags for filtering',
                ],
                'project' => [
                    'type' => 'string',
                    'description' => 'Repo or project name (null for cross-project)',
                ],
                'confidence' => [
                    'type' => 'number',
                    'description' => 'Confidence level 0.0-1.0 (default 1.0)',
                ],
                'supersedes' => [
                    'type' => 'string',
                    'description' => 'UUID of an older memory this one replaces',
                ],
                'expires_in' => [
                    'type' => 'integer',
                    'description' => 'Optional TTL in hours (for session-scoped context)',
                ],
            ],
            'required' => ['content', 'type'],
        ];
    }

    public function handle(array $args, array $context = []): array
    {
        try {
            $content = $this->requireString($args, 'content', 50000);
            $type = $this->requireEnum($args, 'type', BrainMemory::VALID_TYPES);
        } catch (\InvalidArgumentException $e) {
            return $this->error($e->getMessage());
        }

        $workspaceId = $context['workspace_id'] ?? null;
        if ($workspaceId === null) {
            return $this->error('workspace_id is required');
        }

        $agentId = $context['agent_id'] ?? 'unknown';

        $expiresAt = null;
        if (! empty($args['expires_in'])) {
            $expiresAt = now()->addHours((int) $args['expires_in']);
        }

        return $this->withCircuitBreaker('brain', function () use ($args, $content, $type, $workspaceId, $agentId, $expiresAt) {
            $memory = BrainMemory::create([
                'workspace_id' => $workspaceId,
                'agent_id' => $agentId,
                'type' => $type,
                'content' => $content,
                'tags' => $args['tags'] ?? [],
                'project' => $args['project'] ?? null,
                'confidence' => $args['confidence'] ?? 1.0,
                'supersedes_id' => $args['supersedes'] ?? null,
                'expires_at' => $expiresAt,
            ]);

            /** @var BrainService $brainService */
            $brainService = app(BrainService::class);
            $brainService->remember($memory);

            return $this->success([
                'id' => $memory->id,
                'type' => $memory->type,
                'agent_id' => $memory->agent_id,
                'project' => $memory->project,
                'supersedes' => $memory->supersedes_id,
            ]);
        }, fn () => $this->error('Brain service temporarily unavailable', 'service_unavailable'));
    }
}
```

**Step 4: Run tests to verify they pass**

Run: `cd /Users/snider/Code/php-agentic && ./vendor/bin/pest tests/Unit/Tools/BrainRememberTest.php`
Expected: PASS

**Step 5: Commit**

```bash
cd /Users/snider/Code/php-agentic
git add Mcp/Tools/Agent/Brain/BrainRemember.php tests/Unit/Tools/BrainRememberTest.php
git commit -m "feat(brain): add brain_remember MCP tool"
```

---

### Task 4: BrainRecall MCP Tool

**Files:**
- Create: `Mcp/Tools/Agent/Brain/BrainRecall.php`
- Create: `tests/Unit/Tools/BrainRecallTest.php`

**Step 1: Write the failing test**

```php
<?php

declare(strict_types=1);

use Core\Mod\Agentic\Mcp\Tools\Agent\Brain\BrainRecall;

it('has correct name and category', function () {
    $tool = new BrainRecall();
    expect($tool->name())->toBe('brain_recall');
    expect($tool->category())->toBe('brain');
});

it('requires read scope', function () {
    $tool = new BrainRecall();
    expect($tool->requiredScopes())->toContain('read');
});

it('requires query in input schema', function () {
    $tool = new BrainRecall();
    $schema = $tool->inputSchema();
    expect($schema['required'])->toContain('query');
});

it('returns error when query is missing', function () {
    $tool = new BrainRecall();
    $result = $tool->handle([], ['workspace_id' => 1]);
    expect($result)->toHaveKey('error');
});

it('returns error when workspace_id is missing', function () {
    $tool = new BrainRecall();
    $result = $tool->handle(['query' => 'test'], []);
    expect($result)->toHaveKey('error');
});
```

**Step 2: Run test to verify it fails**

Run: `cd /Users/snider/Code/php-agentic && ./vendor/bin/pest tests/Unit/Tools/BrainRecallTest.php`
Expected: FAIL — class not found

**Step 3: Write the tool**

```php
<?php

declare(strict_types=1);

namespace Core\Mod\Agentic\Mcp\Tools\Agent\Brain;

use Core\Mcp\Dependencies\ToolDependency;
use Core\Mod\Agentic\Mcp\Tools\Agent\AgentTool;
use Core\Mod\Agentic\Models\BrainMemory;
use Core\Mod\Agentic\Services\BrainService;

class BrainRecall extends AgentTool
{
    protected string $category = 'brain';
    protected array $scopes = ['read'];

    public function dependencies(): array
    {
        return [
            ToolDependency::contextExists('workspace_id', 'Workspace context required'),
        ];
    }

    public function name(): string
    {
        return 'brain_recall';
    }

    public function description(): string
    {
        return 'Semantic search across the shared agent knowledge graph. Returns the most relevant memories for a natural language query.';
    }

    public function inputSchema(): array
    {
        return [
            'type' => 'object',
            'properties' => [
                'query' => [
                    'type' => 'string',
                    'description' => 'Natural language query (e.g. "How does verdict classification work?")',
                ],
                'top_k' => [
                    'type' => 'integer',
                    'description' => 'Number of results to return (default 5, max 20)',
                ],
                'filter' => [
                    'type' => 'object',
                    'description' => 'Optional filters to narrow search',
                    'properties' => [
                        'project' => [
                            'type' => 'string',
                            'description' => 'Filter by project name',
                        ],
                        'type' => [
                            'type' => 'array',
                            'items' => ['type' => 'string'],
                            'description' => 'Filter by memory types',
                        ],
                        'agent_id' => [
                            'type' => 'string',
                            'description' => 'Filter by agent who created the memory',
                        ],
                        'min_confidence' => [
                            'type' => 'number',
                            'description' => 'Minimum confidence threshold',
                        ],
                    ],
                ],
            ],
            'required' => ['query'],
        ];
    }

    public function handle(array $args, array $context = []): array
    {
        try {
            $query = $this->requireString($args, 'query', 2000);
        } catch (\InvalidArgumentException $e) {
            return $this->error($e->getMessage());
        }

        $workspaceId = $context['workspace_id'] ?? null;
        if ($workspaceId === null) {
            return $this->error('workspace_id is required');
        }

        $topK = min($this->optionalInt($args, 'top_k', 5, 1, 20) ?? 5, 20);
        $filter = $args['filter'] ?? [];

        return $this->withCircuitBreaker('brain', function () use ($query, $topK, $filter, $workspaceId) {
            /** @var BrainService $brainService */
            $brainService = app(BrainService::class);
            $results = $brainService->recall($query, $topK, $filter, $workspaceId);

            return $this->success([
                'count' => count($results['memories']),
                'memories' => array_map(function ($memory) use ($results) {
                    $memory['similarity'] = $results['scores'][$memory['id']] ?? 0;

                    return $memory;
                }, $results['memories']),
            ]);
        }, fn () => $this->error('Brain service temporarily unavailable', 'service_unavailable'));
    }
}
```

**Step 4: Run tests to verify they pass**

Run: `cd /Users/snider/Code/php-agentic && ./vendor/bin/pest tests/Unit/Tools/BrainRecallTest.php`
Expected: PASS

**Step 5: Commit**

```bash
cd /Users/snider/Code/php-agentic
git add Mcp/Tools/Agent/Brain/BrainRecall.php tests/Unit/Tools/BrainRecallTest.php
git commit -m "feat(brain): add brain_recall MCP tool"
```

---

### Task 5: BrainForget + BrainList MCP Tools

**Files:**
- Create: `Mcp/Tools/Agent/Brain/BrainForget.php`
- Create: `Mcp/Tools/Agent/Brain/BrainList.php`
- Create: `tests/Unit/Tools/BrainForgetTest.php`
- Create: `tests/Unit/Tools/BrainListTest.php`

**Step 1: Write the failing tests**

`tests/Unit/Tools/BrainForgetTest.php`:
```php
<?php

declare(strict_types=1);

use Core\Mod\Agentic\Mcp\Tools\Agent\Brain\BrainForget;

it('has correct name and category', function () {
    $tool = new BrainForget();
    expect($tool->name())->toBe('brain_forget');
    expect($tool->category())->toBe('brain');
});

it('requires write scope', function () {
    $tool = new BrainForget();
    expect($tool->requiredScopes())->toContain('write');
});

it('requires id in input schema', function () {
    $tool = new BrainForget();
    $schema = $tool->inputSchema();
    expect($schema['required'])->toContain('id');
});
```

`tests/Unit/Tools/BrainListTest.php`:
```php
<?php

declare(strict_types=1);

use Core\Mod\Agentic\Mcp\Tools\Agent\Brain\BrainList;

it('has correct name and category', function () {
    $tool = new BrainList();
    expect($tool->name())->toBe('brain_list');
    expect($tool->category())->toBe('brain');
});

it('requires read scope', function () {
    $tool = new BrainList();
    expect($tool->requiredScopes())->toContain('read');
});

it('returns error when workspace_id is missing', function () {
    $tool = new BrainList();
    $result = $tool->handle([], []);
    expect($result)->toHaveKey('error');
});
```

**Step 2: Run tests to verify they fail**

Run: `cd /Users/snider/Code/php-agentic && ./vendor/bin/pest tests/Unit/Tools/BrainForgetTest.php tests/Unit/Tools/BrainListTest.php`
Expected: FAIL

**Step 3: Write BrainForget**

```php
<?php

declare(strict_types=1);

namespace Core\Mod\Agentic\Mcp\Tools\Agent\Brain;

use Core\Mcp\Dependencies\ToolDependency;
use Core\Mod\Agentic\Mcp\Tools\Agent\AgentTool;
use Core\Mod\Agentic\Models\BrainMemory;
use Core\Mod\Agentic\Services\BrainService;

class BrainForget extends AgentTool
{
    protected string $category = 'brain';
    protected array $scopes = ['write'];

    public function dependencies(): array
    {
        return [
            ToolDependency::contextExists('workspace_id', 'Workspace context required'),
        ];
    }

    public function name(): string
    {
        return 'brain_forget';
    }

    public function description(): string
    {
        return 'Soft-delete a memory from the knowledge graph. Keeps audit trail but removes from search.';
    }

    public function inputSchema(): array
    {
        return [
            'type' => 'object',
            'properties' => [
                'id' => [
                    'type' => 'string',
                    'description' => 'UUID of the memory to forget',
                ],
                'reason' => [
                    'type' => 'string',
                    'description' => 'Why this memory is being removed',
                ],
            ],
            'required' => ['id'],
        ];
    }

    public function handle(array $args, array $context = []): array
    {
        try {
            $id = $this->requireString($args, 'id');
        } catch (\InvalidArgumentException $e) {
            return $this->error($e->getMessage());
        }

        $workspaceId = $context['workspace_id'] ?? null;
        if ($workspaceId === null) {
            return $this->error('workspace_id is required');
        }

        return $this->withCircuitBreaker('brain', function () use ($id, $workspaceId) {
            $memory = BrainMemory::forWorkspace($workspaceId)->find($id);

            if (! $memory) {
                return $this->error("Memory not found: {$id}");
            }

            /** @var BrainService $brainService */
            $brainService = app(BrainService::class);
            $brainService->forget($id);

            return $this->success([
                'id' => $id,
                'forgotten' => true,
            ]);
        }, fn () => $this->error('Brain service temporarily unavailable', 'service_unavailable'));
    }
}
```

**Step 4: Write BrainList**

```php
<?php

declare(strict_types=1);

namespace Core\Mod\Agentic\Mcp\Tools\Agent\Brain;

use Core\Mcp\Dependencies\ToolDependency;
use Core\Mod\Agentic\Mcp\Tools\Agent\AgentTool;
use Core\Mod\Agentic\Models\BrainMemory;

class BrainList extends AgentTool
{
    protected string $category = 'brain';
    protected array $scopes = ['read'];

    public function dependencies(): array
    {
        return [
            ToolDependency::contextExists('workspace_id', 'Workspace context required'),
        ];
    }

    public function name(): string
    {
        return 'brain_list';
    }

    public function description(): string
    {
        return 'Browse memories by type, project, or agent. No vector search — pure relational filter for auditing and exploration.';
    }

    public function inputSchema(): array
    {
        return [
            'type' => 'object',
            'properties' => [
                'project' => [
                    'type' => 'string',
                    'description' => 'Filter by project name',
                ],
                'type' => [
                    'type' => 'string',
                    'enum' => BrainMemory::VALID_TYPES,
                    'description' => 'Filter by memory type',
                ],
                'agent_id' => [
                    'type' => 'string',
                    'description' => 'Filter by agent who created the memory',
                ],
                'limit' => [
                    'type' => 'integer',
                    'description' => 'Max results (default 20, max 100)',
                ],
            ],
            'required' => [],
        ];
    }

    public function handle(array $args, array $context = []): array
    {
        $workspaceId = $context['workspace_id'] ?? null;
        if ($workspaceId === null) {
            return $this->error('workspace_id is required');
        }

        $limit = min($this->optionalInt($args, 'limit', 20, 1, 100) ?? 20, 100);

        $query = BrainMemory::forWorkspace($workspaceId)
            ->active()
            ->latestVersions();

        if (! empty($args['project'])) {
            $query->forProject($args['project']);
        }

        if (! empty($args['type'])) {
            $query->ofType($args['type']);
        }

        if (! empty($args['agent_id'])) {
            $query->byAgent($args['agent_id']);
        }

        $memories = $query->orderByDesc('created_at')
            ->limit($limit)
            ->get();

        return $this->success([
            'count' => $memories->count(),
            'memories' => $memories->map(fn (BrainMemory $m) => $m->toMcpContext())->all(),
        ]);
    }
}
```

**Step 5: Run tests to verify they pass**

Run: `cd /Users/snider/Code/php-agentic && ./vendor/bin/pest tests/Unit/Tools/`
Expected: PASS

**Step 6: Commit**

```bash
cd /Users/snider/Code/php-agentic
git add Mcp/Tools/Agent/Brain/ tests/Unit/Tools/BrainForgetTest.php tests/Unit/Tools/BrainListTest.php
git commit -m "feat(brain): add brain_forget and brain_list MCP tools"
```

---

### Task 6: Register Brain Tools + Config

**Files:**
- Modify: `Boot.php`
- Modify: `config.php`

**Step 1: Add BrainService config**

Add to `config.php`:

```php
'brain' => [
    'ollama_url' => env('BRAIN_OLLAMA_URL', 'http://localhost:11434'),
    'qdrant_url' => env('BRAIN_QDRANT_URL', 'http://localhost:6334'),
    'collection' => env('BRAIN_COLLECTION', 'openbrain'),
],
```

**Step 2: Register BrainService singleton in Boot.php**

In the `register()` method, add:

```php
$this->app->singleton(\Core\Mod\Agentic\Services\BrainService::class, function ($app) {
    return new \Core\Mod\Agentic\Services\BrainService(
        ollamaUrl: config('mcp.brain.ollama_url', 'http://localhost:11434'),
        qdrantUrl: config('mcp.brain.qdrant_url', 'http://localhost:6334'),
        collection: config('mcp.brain.collection', 'openbrain'),
    );
});
```

**Step 3: Register brain tools in the AgentToolRegistry**

The tools are auto-discovered by the registry when registered. In `Boot.php`, update the `onMcpTools` method or add brain tool registration wherever Session/Plan/State tools are registered. Check how existing tools are registered — likely in the MCP module's boot, not here. If tools are registered elsewhere, add them there.

Look at how Session/Plan tools are registered:

```bash
cd /Users/snider/Code/php-agentic && grep -r "BrainRemember\|SessionStart\|register.*Tool" Boot.php Mcp/ --include="*.php" -l
```

Follow the same pattern for the 4 brain tools:

```php
$registry->registerMany([
    new \Core\Mod\Agentic\Mcp\Tools\Agent\Brain\BrainRemember(),
    new \Core\Mod\Agentic\Mcp\Tools\Agent\Brain\BrainRecall(),
    new \Core\Mod\Agentic\Mcp\Tools\Agent\Brain\BrainForget(),
    new \Core\Mod\Agentic\Mcp\Tools\Agent\Brain\BrainList(),
]);
```

**Step 4: Commit**

```bash
cd /Users/snider/Code/php-agentic
git add Boot.php config.php
git commit -m "feat(brain): register BrainService and brain tools"
```

---

### Task 7: Go Brain Bridge Subsystem

**Files:**
- Create: `/Users/snider/Code/go-ai/mcp/brain/brain.go`
- Create: `/Users/snider/Code/go-ai/mcp/brain/tools.go`
- Create: `/Users/snider/Code/go-ai/mcp/brain/brain_test.go`

**Step 1: Write the failing test**

`brain_test.go`:
```go
package brain

import (
	"testing"
)

func TestSubsystem_Name(t *testing.T) {
	sub := New(nil)
	if sub.Name() != "brain" {
		t.Errorf("Name() = %q, want %q", sub.Name(), "brain")
	}
}

func TestBuildRememberMessage(t *testing.T) {
	msg := buildBridgeMessage("brain_remember", map[string]any{
		"content": "test memory",
		"type":    "observation",
	})
	if msg.Type != "brain_remember" {
		t.Errorf("Type = %q, want %q", msg.Type, "brain_remember")
	}
	if msg.Channel != "brain:remember" {
		t.Errorf("Channel = %q, want %q", msg.Channel, "brain:remember")
	}
}
```

**Step 2: Run test to verify it fails**

Run: `cd /Users/snider/Code/go-ai && go test ./mcp/brain/ -v`
Expected: FAIL — package not found

**Step 3: Write the subsystem**

`brain.go`:
```go
package brain

import (
	"context"
	"time"

	"dappco.re/go/ai/mcp/ide"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// Subsystem bridges brain_* MCP tools to the Laravel backend.
type Subsystem struct {
	bridge *ide.Bridge
}

// New creates a brain subsystem using an existing IDE bridge.
func New(bridge *ide.Bridge) *Subsystem {
	return &Subsystem{bridge: bridge}
}

// Name implements mcp.Subsystem.
func (s *Subsystem) Name() string { return "brain" }

// RegisterTools implements mcp.Subsystem.
func (s *Subsystem) RegisterTools(server *mcp.Server) {
	s.registerTools(server)
}

// Shutdown implements mcp.SubsystemWithShutdown.
func (s *Subsystem) Shutdown(_ context.Context) error { return nil }

func buildBridgeMessage(toolName string, data any) ide.BridgeMessage {
	channelMap := map[string]string{
		"brain_remember": "brain:remember",
		"brain_recall":   "brain:recall",
		"brain_forget":   "brain:forget",
		"brain_list":     "brain:list",
	}
	return ide.BridgeMessage{
		Type:      toolName,
		Channel:   channelMap[toolName],
		Data:      data,
		Timestamp: time.Now(),
	}
}
```

`tools.go`:
```go
package brain

import (
	"context"
	"errors"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

var errBridgeNotAvailable = errors.New("brain: Laravel bridge not connected")

// Input/output types

type RememberInput struct {
	Content    string   `json:"content"`
	Type       string   `json:"type"`
	Tags       []string `json:"tags,omitempty"`
	Project    string   `json:"project,omitempty"`
	Confidence float64  `json:"confidence,omitempty"`
	Supersedes string   `json:"supersedes,omitempty"`
	ExpiresIn  int      `json:"expires_in,omitempty"`
}

type RememberOutput struct {
	Sent      bool      `json:"sent"`
	Timestamp time.Time `json:"timestamp"`
}

type RecallInput struct {
	Query string         `json:"query"`
	TopK  int            `json:"top_k,omitempty"`
	Filter map[string]any `json:"filter,omitempty"`
}

type RecallOutput struct {
	Sent      bool      `json:"sent"`
	Timestamp time.Time `json:"timestamp"`
}

type ForgetInput struct {
	ID     string `json:"id"`
	Reason string `json:"reason,omitempty"`
}

type ForgetOutput struct {
	Sent      bool      `json:"sent"`
	Timestamp time.Time `json:"timestamp"`
}

type ListInput struct {
	Project string `json:"project,omitempty"`
	Type    string `json:"type,omitempty"`
	AgentID string `json:"agent_id,omitempty"`
	Limit   int    `json:"limit,omitempty"`
}

type ListOutput struct {
	Sent      bool      `json:"sent"`
	Timestamp time.Time `json:"timestamp"`
}

func (s *Subsystem) registerTools(server *mcp.Server) {
	mcp.AddTool(server, &mcp.Tool{
		Name:        "brain_remember",
		Description: "Store a memory in the shared agent knowledge graph",
	}, s.remember)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "brain_recall",
		Description: "Semantic search across the shared agent knowledge graph",
	}, s.recall)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "brain_forget",
		Description: "Soft-delete a memory from the knowledge graph",
	}, s.forget)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "brain_list",
		Description: "Browse memories by type, project, or agent",
	}, s.list)
}

func (s *Subsystem) remember(_ context.Context, _ *mcp.CallToolRequest, input RememberInput) (*mcp.CallToolResult, RememberOutput, error) {
	if s.bridge == nil {
		return nil, RememberOutput{}, errBridgeNotAvailable
	}
	err := s.bridge.Send(buildBridgeMessage("brain_remember", input))
	if err != nil {
		return nil, RememberOutput{}, err
	}
	return nil, RememberOutput{Sent: true, Timestamp: time.Now()}, nil
}

func (s *Subsystem) recall(_ context.Context, _ *mcp.CallToolRequest, input RecallInput) (*mcp.CallToolResult, RecallOutput, error) {
	if s.bridge == nil {
		return nil, RecallOutput{}, errBridgeNotAvailable
	}
	err := s.bridge.Send(buildBridgeMessage("brain_recall", input))
	if err != nil {
		return nil, RecallOutput{}, err
	}
	return nil, RecallOutput{Sent: true, Timestamp: time.Now()}, nil
}

func (s *Subsystem) forget(_ context.Context, _ *mcp.CallToolRequest, input ForgetInput) (*mcp.CallToolResult, ForgetOutput, error) {
	if s.bridge == nil {
		return nil, ForgetOutput{}, errBridgeNotAvailable
	}
	err := s.bridge.Send(buildBridgeMessage("brain_forget", input))
	if err != nil {
		return nil, ForgetOutput{}, err
	}
	return nil, ForgetOutput{Sent: true, Timestamp: time.Now()}, nil
}

func (s *Subsystem) list(_ context.Context, _ *mcp.CallToolRequest, input ListInput) (*mcp.CallToolResult, ListOutput, error) {
	if s.bridge == nil {
		return nil, ListOutput{}, errBridgeNotAvailable
	}
	err := s.bridge.Send(buildBridgeMessage("brain_list", input))
	if err != nil {
		return nil, ListOutput{}, err
	}
	return nil, ListOutput{Sent: true, Timestamp: time.Now()}, nil
}
```

**Step 4: Run tests to verify they pass**

Run: `cd /Users/snider/Code/go-ai && go test ./mcp/brain/ -v`
Expected: PASS

**Step 5: Register subsystem in the MCP service**

Find where the IDE subsystem is registered (likely in the CLI or main entry point) and add brain alongside it:

```go
brainSub := brain.New(ideSub.Bridge())
mcpSvc, err := mcp.New(
    mcp.WithSubsystem(ideSub),
    mcp.WithSubsystem(brainSub),
)
```

**Step 6: Commit**

```bash
cd /Users/snider/Code/go-ai
git add mcp/brain/
git commit -m "feat(brain): add Go brain bridge subsystem for OpenBrain MCP tools"
```

---

### Task 8: MEMORY.md Migration Seed Script

**Files:**
- Create: `Console/Commands/BrainSeedFromMemoryFiles.php`

**Step 1: Write the artisan command**

```php
<?php

declare(strict_types=1);

namespace Core\Mod\Agentic\Console\Commands;

use Core\Mod\Agentic\Models\BrainMemory;
use Core\Mod\Agentic\Services\BrainService;
use Illuminate\Console\Command;
use Illuminate\Support\Facades\File;
use Illuminate\Support\Str;

class BrainSeedFromMemoryFiles extends Command
{
    protected $signature = 'brain:seed-memory
        {path? : Base path to scan for MEMORY.md files (default: ~/.claude/projects)}
        {--workspace= : Workspace ID to assign memories to}
        {--agent=virgil : Agent ID for these memories}
        {--dry-run : Show what would be imported without importing}';

    protected $description = 'Import MEMORY.md files from Claude Code worktrees into OpenBrain';

    public function handle(BrainService $brainService): int
    {
        $basePath = $this->argument('path')
            ?? rtrim($_SERVER['HOME'] ?? '', '/').'/.claude/projects';

        $workspaceId = $this->option('workspace');
        if (! $workspaceId) {
            $this->error('--workspace is required');

            return self::FAILURE;
        }

        $agentId = $this->option('agent');
        $dryRun = $this->option('dry-run');

        $brainService->ensureCollection();

        $files = $this->findMemoryFiles($basePath);
        $this->info("Found ".count($files)." MEMORY.md files");

        $imported = 0;

        foreach ($files as $file) {
            $content = File::get($file);
            $projectName = $this->guessProject($file);
            $sections = $this->parseSections($content);

            foreach ($sections as $section) {
                if (strlen(trim($section['content'])) < 20) {
                    continue;
                }

                if ($dryRun) {
                    $this->line("[DRY RUN] Would import: {$section['title']} (project: {$projectName})");

                    continue;
                }

                $memory = BrainMemory::create([
                    'workspace_id' => (int) $workspaceId,
                    'agent_id' => $agentId,
                    'type' => $this->guessType($section['title']),
                    'content' => "## {$section['title']}\n\n{$section['content']}",
                    'tags' => $this->extractTags($section['content']),
                    'project' => $projectName,
                    'confidence' => 0.8,
                ]);

                $brainService->remember($memory);
                $imported++;
                $this->line("Imported: {$section['title']} (project: {$projectName})");
            }
        }

        $this->info("Imported {$imported} memories into OpenBrain");

        return self::SUCCESS;
    }

    private function findMemoryFiles(string $basePath): array
    {
        $files = [];

        if (! is_dir($basePath)) {
            return $files;
        }

        $iterator = new \RecursiveIteratorIterator(
            new \RecursiveDirectoryIterator($basePath, \FilesystemIterator::SKIP_DOTS),
            \RecursiveIteratorIterator::LEAVES_ONLY
        );

        foreach ($iterator as $file) {
            if ($file->getFilename() === 'MEMORY.md' || Str::endsWith($file->getPathname(), '/memory/MEMORY.md')) {
                $files[] = $file->getPathname();
            }
        }

        return $files;
    }

    private function guessProject(string $filepath): ?string
    {
        if (preg_match('#/projects/-Users-\w+-Code-([^/]+)/#', $filepath, $m)) {
            return $m[1];
        }

        return null;
    }

    private function guessType(string $title): string
    {
        $lower = strtolower($title);

        if (Str::contains($lower, ['decision', 'chose', 'approach'])) {
            return BrainMemory::TYPE_DECISION;
        }
        if (Str::contains($lower, ['architecture', 'stack', 'infrastructure'])) {
            return BrainMemory::TYPE_ARCHITECTURE;
        }
        if (Str::contains($lower, ['convention', 'rule', 'standard', 'pattern'])) {
            return BrainMemory::TYPE_CONVENTION;
        }
        if (Str::contains($lower, ['bug', 'fix', 'issue', 'error'])) {
            return BrainMemory::TYPE_BUG;
        }
        if (Str::contains($lower, ['plan', 'todo', 'roadmap'])) {
            return BrainMemory::TYPE_PLAN;
        }
        if (Str::contains($lower, ['research', 'finding', 'analysis'])) {
            return BrainMemory::TYPE_RESEARCH;
        }

        return BrainMemory::TYPE_OBSERVATION;
    }

    private function extractTags(string $content): array
    {
        $tags = [];

        // Extract backtick-quoted identifiers as potential tags
        if (preg_match_all('/`([a-z][a-z0-9_-]+)`/', $content, $matches)) {
            $tags = array_unique(array_slice($matches[1], 0, 10));
        }

        return array_values($tags);
    }

    private function parseSections(string $content): array
    {
        $sections = [];
        $lines = explode("\n", $content);
        $currentTitle = null;
        $currentContent = [];

        foreach ($lines as $line) {
            if (preg_match('/^#{1,3}\s+(.+)$/', $line, $m)) {
                if ($currentTitle !== null) {
                    $sections[] = [
                        'title' => $currentTitle,
                        'content' => trim(implode("\n", $currentContent)),
                    ];
                }
                $currentTitle = $m[1];
                $currentContent = [];
            } else {
                $currentContent[] = $line;
            }
        }

        if ($currentTitle !== null) {
            $sections[] = [
                'title' => $currentTitle,
                'content' => trim(implode("\n", $currentContent)),
            ];
        }

        return $sections;
    }
}
```

**Step 2: Register the command**

In `Boot.php`, the `onConsole` method (or `ConsoleBooting` listener) should register:

```php
$this->commands([
    \Core\Mod\Agentic\Console\Commands\BrainSeedFromMemoryFiles::class,
]);
```

**Step 3: Test with dry run**

Run: `php artisan brain:seed-memory --workspace=1 --dry-run`
Expected: Lists found MEMORY.md files and sections without importing

**Step 4: Commit**

```bash
cd /Users/snider/Code/php-agentic
git add Console/Commands/BrainSeedFromMemoryFiles.php Boot.php
git commit -m "feat(brain): add brain:seed-memory command for MEMORY.md migration"
```

---

## Summary

| Task | Component | Files | Commit |
|------|-----------|-------|--------|
| 1 | Migration + Model | 2 created | `feat(brain): add BrainMemory model and migration` |
| 2 | BrainService | 2 created | `feat(brain): add BrainService with Ollama + Qdrant` |
| 3 | brain_remember tool | 2 created | `feat(brain): add brain_remember MCP tool` |
| 4 | brain_recall tool | 2 created | `feat(brain): add brain_recall MCP tool` |
| 5 | brain_forget + brain_list | 4 created | `feat(brain): add brain_forget and brain_list MCP tools` |
| 6 | Registration + config | 2 modified | `feat(brain): register BrainService and brain tools` |
| 7 | Go bridge subsystem | 3 created | `feat(brain): add Go brain bridge subsystem` |
| 8 | MEMORY.md migration | 1 created, 1 modified | `feat(brain): add brain:seed-memory command` |

**Total: 18 files across 2 repos, 8 commits.**
