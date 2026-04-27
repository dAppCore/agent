<?php

// SPDX-License-Identifier: EUPL-1.2

declare(strict_types=1);

use Core\Mod\Agentic\Models\AgentApiKey;
use Core\Mod\Agentic\Services\BrainService;
use Illuminate\Http\Client\Request;
use Illuminate\Support\Facades\Http;

function brainTagsScopesRegisterRoutes(): void
{
    require __DIR__.'/../../../Routes/api.php';
}

function brainTagsScopesKey(): string
{
    $workspace = createWorkspace();
    $key = createApiKey($workspace, 'Brain Reader', [AgentApiKey::PERM_BRAIN_READ], 1000);

    return (string) $key->plainTextKey;
}

beforeEach(function (): void {
    brainTagsScopesRegisterRoutes();

    $this->app->instance(BrainService::class, new BrainService(
        ollamaUrl: 'https://ollama.test',
        qdrantUrl: 'https://qdrant.test',
        collection: 'openbrain',
        embeddingModel: 'embeddinggemma',
        verifySsl: false,
        elasticsearchUrl: 'https://elasticsearch.test',
    ));
});

test('BrainController_tags_Good_returns_tag_counts_from_elasticsearch_terms_aggregation', function (): void {
    Http::fake([
        'https://elasticsearch.test/brain_memories/_search' => Http::response([
            'aggregations' => [
                'tags' => [
                    'buckets' => [
                        ['key' => 'architecture', 'doc_count' => 7],
                        ['key' => 'openbrain', 'doc_count' => 3],
                    ],
                ],
            ],
        ]),
    ]);

    $response = $this
        ->withHeader('Authorization', 'Bearer '.brainTagsScopesKey())
        ->getJson('/v1/brain/tags');

    $response
        ->assertOk()
        ->assertExactJson([
            'data' => [
                'architecture' => 7,
                'openbrain' => 3,
            ],
        ]);

    Http::assertSent(fn (Request $request): bool => $request->url() === 'https://elasticsearch.test/brain_memories/_search'
        && $request->method() === 'POST'
        && $request['size'] === 0
        && $request['aggs'] === [
            'tags' => [
                'terms' => [
                    'field' => 'tags.keyword',
                    'size' => 1000,
                ],
            ],
        ]);
});

test('BrainController_tags_Bad_returns_service_error_when_elasticsearch_fails', function (): void {
    Http::fake([
        'https://elasticsearch.test/brain_memories/_search' => Http::response(['error' => 'unavailable'], 503),
    ]);

    $response = $this
        ->withHeader('Authorization', 'Bearer '.brainTagsScopesKey())
        ->getJson('/v1/brain/tags');

    $response
        ->assertStatus(503)
        ->assertExactJson([
            'error' => 'service_error',
            'message' => 'Brain service temporarily unavailable.',
        ]);
});

test('BrainController_tags_Ugly_ignores_malformed_tag_buckets', function (): void {
    Http::fake([
        'https://elasticsearch.test/brain_memories/_search' => Http::response([
            'aggregations' => [
                'tags' => [
                    'buckets' => [
                        ['key' => 'indexed', 'doc_count' => 4],
                        ['key' => ['not-a-string'], 'doc_count' => 9],
                        ['doc_count' => 2],
                    ],
                ],
            ],
        ]),
    ]);

    $response = $this
        ->withHeader('Authorization', 'Bearer '.brainTagsScopesKey())
        ->getJson('/v1/brain/tags');

    $response
        ->assertOk()
        ->assertExactJson([
            'data' => [
                'indexed' => 4,
            ],
        ]);
});

test('BrainController_scopes_Good_returns_hierarchical_scope_tree_from_composite_aggregation', function (): void {
    Http::fake([
        'https://elasticsearch.test/brain_memories/_search' => Http::response([
            'aggregations' => [
                'scopes' => [
                    'buckets' => [
                        ['key' => ['org' => 'core', 'project' => 'agent'], 'doc_count' => 11],
                        ['key' => ['org' => 'core', 'project' => 'host'], 'doc_count' => 5],
                        ['key' => ['org' => 'ops', 'project' => 'deploy'], 'doc_count' => 2],
                    ],
                ],
            ],
        ]),
    ]);

    $response = $this
        ->withHeader('Authorization', 'Bearer '.brainTagsScopesKey())
        ->getJson('/v1/brain/scopes');

    $response
        ->assertOk()
        ->assertExactJson([
            'data' => [
                'core' => [
                    'agent' => 11,
                    'host' => 5,
                ],
                'ops' => [
                    'deploy' => 2,
                ],
            ],
        ]);

    Http::assertSent(fn (Request $request): bool => $request->url() === 'https://elasticsearch.test/brain_memories/_search'
        && $request->method() === 'POST'
        && $request['size'] === 0
        && $request['aggs'] === [
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
        ]);
});

test('BrainController_scopes_Bad_returns_service_error_when_elasticsearch_fails', function (): void {
    Http::fake([
        'https://elasticsearch.test/brain_memories/_search' => Http::response(['error' => 'unavailable'], 500),
    ]);

    $response = $this
        ->withHeader('Authorization', 'Bearer '.brainTagsScopesKey())
        ->getJson('/v1/brain/scopes');

    $response
        ->assertStatus(503)
        ->assertExactJson([
            'error' => 'service_error',
            'message' => 'Brain service temporarily unavailable.',
        ]);
});

test('BrainController_scopes_Ugly_ignores_incomplete_scope_buckets', function (): void {
    Http::fake([
        'https://elasticsearch.test/brain_memories/_search' => Http::response([
            'aggregations' => [
                'scopes' => [
                    'buckets' => [
                        ['key' => ['org' => 'core', 'project' => 'agent'], 'doc_count' => 3],
                        ['key' => ['org' => 'core'], 'doc_count' => 8],
                        ['key' => ['project' => 'missing-org'], 'doc_count' => 4],
                        ['doc_count' => 1],
                    ],
                ],
            ],
        ]),
    ]);

    $response = $this
        ->withHeader('Authorization', 'Bearer '.brainTagsScopesKey())
        ->getJson('/v1/brain/scopes');

    $response
        ->assertOk()
        ->assertExactJson([
            'data' => [
                'core' => [
                    'agent' => 3,
                ],
            ],
        ]);
});
