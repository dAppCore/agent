<?php

// SPDX-License-Identifier: EUPL-1.2

declare(strict_types=1);

use Core\Mod\Agentic\Models\BrainMemory;
use Core\Mod\Agentic\Services\BrainService;
use Illuminate\Http\Client\Request;
use Illuminate\Support\Carbon;
use Illuminate\Support\Facades\Http;
use Illuminate\Support\Str;

function elasticBrainService(): BrainService
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

function elasticBrainMemory(array $attributes = []): BrainMemory
{
    $memory = new BrainMemory;
    $memory->forceFill(array_merge([
        'id' => Str::uuid()->toString(),
        'workspace_id' => 42,
        'agent_id' => 'virgil',
        'type' => 'architecture',
        'content' => 'Elasticsearch mirrors brain memories for lexical search.',
        'tags' => ['brain', 'search'],
        'project' => 'agent',
        'confidence' => 0.95,
        'indexed_at' => Carbon::parse('2026-04-23 12:00:00', 'UTC'),
    ], $attributes));
    $memory->setAttribute('org', 'core');

    return $memory;
}

test('BrainService_elasticIndex_Good_indexes_memory_document', function (): void {
    $memory = elasticBrainMemory();

    Http::fake([
        "https://elasticsearch.test/brain_memories/_doc/{$memory->id}" => Http::response(['result' => 'updated']),
    ]);

    elasticBrainService()->elasticIndex($memory);

    Http::assertSent(fn (Request $request): bool => $request->url() === "https://elasticsearch.test/brain_memories/_doc/{$memory->id}"
        && $request->method() === 'PUT'
        && $request['id'] === $memory->id
        && $request['content'] === 'Elasticsearch mirrors brain memories for lexical search.'
        && $request['type'] === 'architecture'
        && $request['tags'] === ['brain', 'search']
        && $request['project'] === 'agent'
        && $request['workspace_id'] === 42
        && $request['org'] === 'core'
        && $request['confidence'] === 0.95
        && $request['indexed_at'] === '2026-04-23T12:00:00+00:00');
});

test('BrainService_elasticIndex_Bad_throws_when_indexing_fails', function (): void {
    $memory = elasticBrainMemory();

    Http::fake([
        "https://elasticsearch.test/brain_memories/_doc/{$memory->id}" => Http::response(['error' => 'unavailable'], 500),
    ]);

    expect(fn () => elasticBrainService()->elasticIndex($memory))
        ->toThrow(RuntimeException::class, 'Elasticsearch index failed: 500');
});

test('BrainService_elasticIndex_Ugly_indexes_null_optional_fields', function (): void {
    $memory = elasticBrainMemory([
        'tags' => null,
        'project' => null,
        'indexed_at' => null,
    ]);
    $memory->setAttribute('org', null);

    Http::fake([
        "https://elasticsearch.test/brain_memories/_doc/{$memory->id}" => Http::response(['result' => 'created']),
    ]);

    elasticBrainService()->elasticIndex($memory);

    Http::assertSent(fn (Request $request): bool => $request->url() === "https://elasticsearch.test/brain_memories/_doc/{$memory->id}"
        && $request->method() === 'PUT'
        && $request['tags'] === []
        && $request['project'] === null
        && $request['org'] === null
        && $request['indexed_at'] === null);
});

test('BrainService_elasticDelete_Good_deletes_memory_document', function (): void {
    $memoryId = Str::uuid()->toString();

    Http::fake([
        "https://elasticsearch.test/brain_memories/_doc/{$memoryId}" => Http::response(['result' => 'deleted']),
    ]);

    elasticBrainService()->elasticDelete($memoryId);

    Http::assertSent(fn (Request $request): bool => $request->url() === "https://elasticsearch.test/brain_memories/_doc/{$memoryId}"
        && $request->method() === 'DELETE');
});

test('BrainService_elasticDelete_Bad_throws_when_delete_fails', function (): void {
    $memoryId = Str::uuid()->toString();

    Http::fake([
        "https://elasticsearch.test/brain_memories/_doc/{$memoryId}" => Http::response(['error' => 'unavailable'], 500),
    ]);

    expect(fn () => elasticBrainService()->elasticDelete($memoryId))
        ->toThrow(RuntimeException::class, 'Elasticsearch delete failed: 500');
});

test('BrainService_elasticDelete_Ugly_throws_when_document_is_missing', function (): void {
    $memoryId = Str::uuid()->toString();

    Http::fake([
        "https://elasticsearch.test/brain_memories/_doc/{$memoryId}" => Http::response(['result' => 'not_found'], 404),
    ]);

    expect(fn () => elasticBrainService()->elasticDelete($memoryId))
        ->toThrow(RuntimeException::class, 'Elasticsearch delete failed: 404');
});

test('BrainService_elasticSearch_Good_posts_multi_match_query_with_filters', function (): void {
    Http::fake([
        'https://elasticsearch.test/brain_memories/_search' => Http::response([
            'hits' => [
                'hits' => [
                    ['_id' => 'memory-1', '_score' => 1.25],
                ],
            ],
        ]),
    ]);

    $result = elasticBrainService()->elasticSearch('queue indexing', [
        'workspace_id' => 42,
        'org' => 'core',
        'project' => 'agent',
        'type' => ['architecture', 'pattern'],
        'tags' => 'brain',
        'min_confidence' => 0.7,
    ]);

    expect($result['hits']['hits'][0]['_id'])->toBe('memory-1');

    Http::assertSent(fn (Request $request): bool => $request->url() === 'https://elasticsearch.test/brain_memories/_search'
        && $request->method() === 'POST'
        && $request['query']['bool']['must'][0]['multi_match']['query'] === 'queue indexing'
        && $request['query']['bool']['must'][0]['multi_match']['fields'] === ['content^3', 'type', 'tags', 'project', 'org']
        && $request['query']['bool']['filter'] === [
            ['term' => ['workspace_id' => 42]],
            ['term' => ['org' => 'core']],
            ['term' => ['project' => 'agent']],
            ['terms' => ['type' => ['architecture', 'pattern']]],
            ['term' => ['tags' => 'brain']],
            ['range' => ['confidence' => ['gte' => 0.7]]],
        ]);
});

test('BrainService_elasticSearch_Bad_throws_when_search_fails', function (): void {
    Http::fake([
        'https://elasticsearch.test/brain_memories/_search' => Http::response(['error' => 'unavailable'], 503),
    ]);

    expect(fn () => elasticBrainService()->elasticSearch('queue indexing'))
        ->toThrow(RuntimeException::class, 'Elasticsearch search failed: 503');
});

test('BrainService_elasticSearch_Ugly_uses_match_all_for_empty_query', function (): void {
    Http::fake([
        'https://elasticsearch.test/brain_memories/_search' => Http::response(['hits' => ['hits' => []]]),
    ]);

    elasticBrainService()->elasticSearch('', [
        'tags' => ['brain', 'search'],
    ]);

    Http::assertSent(fn (Request $request): bool => $request->url() === 'https://elasticsearch.test/brain_memories/_search'
        && $request->method() === 'POST'
        && isset($request['query']['bool']['must'][0]['match_all'])
        && $request['query']['bool']['filter'] === [
            ['terms' => ['tags' => ['brain', 'search']]],
        ]);
});
