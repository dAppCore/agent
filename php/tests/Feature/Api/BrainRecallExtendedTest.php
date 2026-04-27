<?php

// SPDX-License-Identifier: EUPL-1.2

declare(strict_types=1);

use Core\Mod\Agentic\Models\AgentApiKey;
use Core\Mod\Agentic\Models\BrainMemory;
use Core\Mod\Agentic\Services\BrainService;
use Illuminate\Http\Client\Request;
use Illuminate\Support\Facades\Http;

function brainRecallExtendedService(): BrainService
{
    return new BrainService(
        ollamaUrl: 'https://ollama.test',
        qdrantUrl: 'https://qdrant.test',
        collection: 'openbrain',
        embeddingModel: 'embeddinggemma',
        verifySsl: false,
        elasticsearchUrl: 'https://elasticsearch.test',
    );
}

function brainRecallExtendedMemory(int $workspaceId, array $attributes = []): BrainMemory
{
    return BrainMemory::create(array_merge([
        'workspace_id' => $workspaceId,
        'agent_id' => 'virgil',
        'type' => 'architecture',
        'content' => 'Hybrid recall keeps semantic memories available.',
        'tags' => ['brain', 'recall'],
        'project' => 'agent',
        'confidence' => 0.95,
        'source' => 'mantis-63',
    ], $attributes));
}

test('BrainController_recall_Good_filters_org_merges_keywords_and_boosts_matches', function (): void {
    config(['mcp.brain.boost_keywords_multiplier' => 1.5]);
    $workspace = createWorkspace();
    $apiKey = createApiKey($workspace, 'Brain Reader', [AgentApiKey::PERM_BRAIN_READ]);
    $vectorMemory = brainRecallExtendedMemory($workspace->id, [
        'content' => 'Vector search finds workspace recall architecture.',
    ]);
    $keywordMemory = brainRecallExtendedMemory($workspace->id, [
        'content' => 'Mantis hybrid keyword recall should win after boost.',
    ]);
    $overlapMemory = brainRecallExtendedMemory($workspace->id, [
        'content' => 'Overlap memory appears in vector and keyword search.',
    ]);

    $this->app->instance(BrainService::class, brainRecallExtendedService());

    Http::fake([
        'https://ollama.test/api/embeddings' => Http::response(['embedding' => array_fill(0, 768, 0.125)]),
        'https://qdrant.test/collections/openbrain/points/search' => Http::response([
            'result' => [
                ['id' => $vectorMemory->id, 'score' => 0.7],
                ['id' => $overlapMemory->id, 'score' => 0.6],
            ],
        ]),
        'https://elasticsearch.test/brain_memories/_search' => Http::response([
            'hits' => [
                'hits' => [
                    ['_id' => $keywordMemory->id, '_score' => 1.0],
                    ['_id' => $overlapMemory->id, '_score' => 0.5],
                ],
            ],
        ]),
    ]);

    $response = $this
        ->withHeader('Authorization', "Bearer {$apiKey->plainTextKey}")
        ->postJson('/v1/brain/recall', [
            'query' => 'hybrid recall',
            'limit' => 2,
            'org' => 'core',
            'project' => 'agent',
            'type' => 'architecture',
            'keywords' => ['hybrid', 'mantis'],
            'boost_keywords' => ['mantis'],
        ]);

    $response->assertOk();
    expect($response->json('data.memories'))->toHaveCount(2)
        ->and($response->json('data.count'))->toBe(2)
        ->and($response->json('data.memories.0.id'))->toBe($keywordMemory->id)
        ->and($response->json('data.memories.0.score'))->toBe(1.5)
        ->and($response->json('data.memories.1.id'))->toBe($vectorMemory->id)
        ->and(array_unique(array_column($response->json('data.memories'), 'id')))->toHaveCount(2);

    Http::assertSent(fn (Request $request): bool => $request->url() === 'https://qdrant.test/collections/openbrain/points/search'
        && $request->method() === 'POST'
        && $request['limit'] === 2
        && $request['filter']['must'] === [
            ['key' => 'workspace_id', 'match' => ['value' => $workspace->id]],
            ['key' => 'org', 'match' => ['value' => 'core']],
            ['key' => 'project', 'match' => ['value' => 'agent']],
            ['key' => 'type', 'match' => ['value' => 'architecture']],
        ]);

    Http::assertSent(fn (Request $request): bool => $request->url() === 'https://elasticsearch.test/brain_memories/_search'
        && $request->method() === 'POST'
        && $request['size'] === 2
        && $request['query']['bool']['must'][0]['multi_match']['query'] === 'hybrid mantis'
        && $request['query']['bool']['filter'] === [
            ['term' => ['workspace_id' => $workspace->id]],
            ['term' => ['org' => 'core']],
            ['term' => ['project' => 'agent']],
            ['term' => ['type' => 'architecture']],
        ]);
});

test('BrainController_recall_Bad_rejects_non_array_keywords', function (): void {
    $workspace = createWorkspace();
    $apiKey = createApiKey($workspace, 'Brain Reader', [AgentApiKey::PERM_BRAIN_READ]);
    $this->app->instance(BrainService::class, brainRecallExtendedService());
    Http::fake();

    $response = $this
        ->withHeader('Authorization', "Bearer {$apiKey->plainTextKey}")
        ->postJson('/v1/brain/recall', [
            'query' => 'hybrid recall',
            'keywords' => 'mantis',
        ]);

    $response->assertStatus(422);
    Http::assertNothingSent();
});

test('BrainController_recall_Ugly_skips_elasticsearch_when_keywords_normalise_empty', function (): void {
    $workspace = createWorkspace();
    $apiKey = createApiKey($workspace, 'Brain Reader', [AgentApiKey::PERM_BRAIN_READ]);
    $memory = brainRecallExtendedMemory($workspace->id);
    $this->app->instance(BrainService::class, brainRecallExtendedService());

    Http::fake([
        'https://ollama.test/api/embeddings' => Http::response(['embedding' => array_fill(0, 768, 0.125)]),
        'https://qdrant.test/collections/openbrain/points/search' => Http::response([
            'result' => [
                ['id' => $memory->id, 'score' => 0.72],
            ],
        ]),
    ]);

    $response = $this
        ->withHeader('Authorization', "Bearer {$apiKey->plainTextKey}")
        ->postJson('/v1/brain/recall', [
            'query' => 'hybrid recall',
            'limit' => 5,
            'keywords' => [' ', ''],
        ]);

    $response->assertOk();
    expect($response->json('data.memories.0.id'))->toBe($memory->id);

    Http::assertNotSent(fn (Request $request): bool => $request->url() === 'https://elasticsearch.test/brain_memories/_search');
});
