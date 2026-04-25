<?php

// SPDX-License-Identifier: EUPL-1.2

declare(strict_types=1);

use Core\Mod\Agentic\Models\AgentApiKey;
use Core\Mod\Agentic\Models\BrainMemory;
use Core\Mod\Agentic\Services\BrainService;
use Core\Tenant\Models\Workspace;
use Illuminate\Http\Client\Request;
use Illuminate\Support\Facades\Http;

beforeEach(function (): void {
    $this->app->instance(BrainService::class, new BrainService(
        ollamaUrl: 'https://ollama.test',
        qdrantUrl: 'https://qdrant.test',
        collection: 'openbrain',
        embeddingModel: 'embeddinggemma',
        verifySsl: false,
        elasticsearchUrl: 'https://elasticsearch.test',
    ));

    require __DIR__.'/../../../Routes/api.php';
});

function brainSearchMemory(Workspace $workspace, array $attributes = []): BrainMemory
{
    return BrainMemory::create(array_merge([
        'workspace_id' => $workspace->id,
        'agent_id' => 'virgil',
        'type' => 'architecture',
        'content' => 'Elasticsearch mirrors brain memories for lexical search.',
        'tags' => ['brain', 'search'],
        'project' => 'agent',
        'confidence' => 0.95,
    ], $attributes));
}

function brainSearchKey(Workspace $workspace, array $permissions = [AgentApiKey::PERM_BRAIN_READ]): AgentApiKey
{
    return createApiKey($workspace, 'Brain Search Key', $permissions);
}

test('BrainController_search_Good_returns_memories_with_elasticsearch_scores', function (): void {
    $workspace = createWorkspace();
    $first = brainSearchMemory($workspace, [
        'content' => 'Queue indexing uses Elasticsearch for keyword recall.',
    ]);
    $second = brainSearchMemory($workspace, [
        'content' => 'Project filters keep search results narrow.',
    ]);
    $key = brainSearchKey($workspace);

    Http::fake([
        'https://elasticsearch.test/brain_memories/_search' => Http::response([
            'hits' => [
                'hits' => [
                    ['_id' => $second->id, '_score' => 2.75],
                    ['_id' => $first->id, '_score' => 1.25],
                ],
            ],
        ]),
    ]);

    $response = $this
        ->withHeader('Authorization', 'Bearer '.$key->plainTextKey)
        ->getJson('/v1/brain/search?q=queue%20indexing&org=core&project=agent');

    $response
        ->assertOk()
        ->assertJsonPath('data.count', 2)
        ->assertJsonPath('data.memories.0.id', $second->id)
        ->assertJsonPath('data.memories.0.score', 2.75)
        ->assertJsonPath('data.memories.1.id', $first->id)
        ->assertJsonPath('data.memories.1.score', 1.25);

    Http::assertSent(fn (Request $request): bool => $request->url() === 'https://elasticsearch.test/brain_memories/_search'
        && $request->method() === 'POST'
        && $request['query']['bool']['must'][0]['multi_match']['query'] === 'queue indexing'
        && $request['query']['bool']['filter'] === [
            ['term' => ['workspace_id' => $workspace->id]],
            ['term' => ['org' => 'core']],
            ['term' => ['project' => 'agent']],
        ]);
});

test('BrainController_search_Bad_returns_service_error_when_elasticsearch_fails', function (): void {
    $workspace = createWorkspace();
    $key = brainSearchKey($workspace);

    Http::fake([
        'https://elasticsearch.test/brain_memories/_search' => Http::response(['error' => 'unavailable'], 503),
    ]);

    $response = $this
        ->withHeader('Authorization', 'Bearer '.$key->plainTextKey)
        ->getJson('/v1/brain/search?q=queue%20indexing');

    $response
        ->assertStatus(503)
        ->assertJsonPath('error', 'service_error')
        ->assertJsonPath('message', 'Brain service temporarily unavailable.');
});

test('BrainController_search_Ugly_limits_results_and_ignores_stale_hits', function (): void {
    $workspace = createWorkspace();
    $otherWorkspace = createWorkspace();
    $visible = brainSearchMemory($workspace, [
        'content' => 'Visible memory should survive stale index entries.',
    ]);
    $hidden = brainSearchMemory($otherWorkspace, [
        'content' => 'Other workspace memory must not leak through ES.',
    ]);
    $key = brainSearchKey($workspace);

    Http::fake([
        'https://elasticsearch.test/brain_memories/_search' => Http::response([
            'hits' => [
                'hits' => [
                    ['_id' => 'missing-memory-id', '_score' => 9.5],
                    ['_id' => $hidden->id, '_score' => 8.5],
                    ['_id' => $visible->id, '_score' => 7.5],
                ],
            ],
        ]),
    ]);

    $response = $this
        ->withHeader('Authorization', 'Bearer '.$key->plainTextKey)
        ->getJson('/v1/brain/search?q=stale&limit=3');

    $response
        ->assertOk()
        ->assertJsonPath('data.count', 1)
        ->assertJsonPath('data.memories.0.id', $visible->id)
        ->assertJsonPath('data.memories.0.score', 7.5);
});
