<?php

// SPDX-License-Identifier: EUPL-1.2

declare(strict_types=1);

use Core\Mod\Agentic\Jobs\EmbedMemory;
use Core\Mod\Agentic\Models\BrainMemory;
use Core\Mod\Agentic\Services\BrainService;
use Illuminate\Http\Client\Request as ClientRequest;
use Illuminate\Support\Facades\Http;
use Illuminate\Support\Facades\Queue;

function rememberValidationBrainService(): BrainService
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

function rememberValidationAttributes(array $attributes = []): array
{
    return array_merge([
        'workspace_id' => $attributes['workspace_id'] ?? createWorkspace()->id,
        'agent_id' => 'virgil',
        'type' => 'observation',
        'content' => 'Brain validation test memory.',
        'tags' => ['brain', 'validation'],
        'confidence' => 0.9,
        'org' => 'core',
        'project' => 'agent',
    ], $attributes);
}

function rememberValidationMemory(array $attributes = []): BrainMemory
{
    return BrainMemory::create(rememberValidationAttributes($attributes));
}

test('BrainRememberValidation_remember_Good_accepts_valid_content_and_tags', function (): void {
    Queue::fake();

    $memory = rememberValidationBrainService()->remember(rememberValidationAttributes([
        'content' => 'hello world',
        'tags' => ['a', 'b'],
    ]));

    expect($memory->exists)->toBeTrue()
        ->and($memory->content)->toBe('hello world')
        ->and($memory->tags)->toBe(['a', 'b']);

    Queue::assertPushed(EmbedMemory::class, fn (EmbedMemory $job): bool => $job->memoryId === $memory->id);
});

test('BrainRememberValidation_remember_Bad_rejects_content_longer_than_65536_bytes', function (): void {
    Queue::fake();

    expect(fn () => rememberValidationBrainService()->remember(rememberValidationAttributes([
        'content' => str_repeat('a', 65537),
    ])))->toThrow(\InvalidArgumentException::class, 'content exceeds maximum length of 65536 bytes');

    Queue::assertNothingPushed();
});

test('BrainRememberValidation_remember_Bad_rejects_more_than_100_tags', function (): void {
    Queue::fake();

    expect(fn () => rememberValidationBrainService()->remember(rememberValidationAttributes([
        'tags' => array_fill(0, 101, 'x'),
    ])))->toThrow(\InvalidArgumentException::class, 'tags array exceeds maximum size of 100');

    Queue::assertNothingPushed();
});

test('BrainRememberValidation_remember_Ugly_rejects_tags_longer_than_128_bytes', function (): void {
    Queue::fake();

    expect(fn () => rememberValidationBrainService()->remember(rememberValidationAttributes([
        'tags' => ['valid', str_repeat('x', 129)],
    ])))->toThrow(\InvalidArgumentException::class, 'tag at index 1 exceeds maximum length of 128');

    Queue::assertNothingPushed();
});

test('BrainRememberValidation_remember_Good_accepts_content_at_exactly_65536_bytes', function (): void {
    Queue::fake();

    $content = str_repeat('a', 65536);
    $memory = rememberValidationBrainService()->remember(rememberValidationAttributes([
        'content' => $content,
        'tags' => ['boundary'],
    ]));

    expect($memory->exists)->toBeTrue()
        ->and($memory->content)->toBe($content);

    Queue::assertPushed(EmbedMemory::class, fn (EmbedMemory $job): bool => $job->memoryId === $memory->id);
});

test('BrainRememberValidation_recall_Bad_rejects_min_confidence_above_one', function (): void {
    expect(fn () => rememberValidationBrainService()->recall(
        'brain validation query',
        5,
        ['min_confidence' => 1.1],
        createWorkspace()->id,
    ))->toThrow(\InvalidArgumentException::class, 'min_confidence must be between 0.0 and 1.0');
});

test('BrainRememberValidation_recall_Bad_rejects_project_filters_longer_than_128_characters', function (): void {
    Http::fake();

    expect(fn () => rememberValidationBrainService()->recall(
        'brain validation query',
        5,
        ['project' => str_repeat('x', 129)],
        createWorkspace()->id,
    ))->toThrow(\InvalidArgumentException::class, 'project exceeds maximum length of 128');

    Http::assertNothingSent();
});

test('BrainRememberValidation_recall_Bad_rejects_agent_id_filters_longer_than_64_characters', function (): void {
    Http::fake();

    expect(fn () => rememberValidationBrainService()->recall(
        'brain validation query',
        5,
        ['agent_id' => str_repeat('x', 65)],
        createWorkspace()->id,
    ))->toThrow(\InvalidArgumentException::class, 'agent_id exceeds maximum length of 64');

    Http::assertNothingSent();
});

test('BrainRememberValidation_recall_Good_accepts_project_filters_within_bounds', function (): void {
    $workspace = createWorkspace();
    Http::fake([
        'https://ollama.test/api/embeddings' => Http::response(['embedding' => array_fill(0, 768, 0.125)]),
        'https://qdrant.test/collections/openbrain/points/search' => Http::response(['result' => []]),
    ]);

    $result = rememberValidationBrainService()->recall(
        'brain validation query',
        5,
        ['project' => 'core'],
        $workspace->id,
    );

    expect($result)->toBe([
        'memories' => [],
        'scores' => [],
    ]);

    Http::assertSent(fn (ClientRequest $request): bool => $request->url() === 'https://qdrant.test/collections/openbrain/points/search'
        && $request->method() === 'POST'
        && $request['filter']['must'] === [
            ['key' => 'workspace_id', 'match' => ['value' => $workspace->id]],
            ['key' => 'project', 'match' => ['value' => 'core']],
        ]);
});

test('BrainRememberValidation_recall_Ugly_accepts_project_filters_at_the_128_character_boundary', function (): void {
    $workspace = createWorkspace();
    $project = str_repeat('x', 128);

    Http::fake([
        'https://ollama.test/api/embeddings' => Http::response(['embedding' => array_fill(0, 768, 0.125)]),
        'https://qdrant.test/collections/openbrain/points/search' => Http::response(['result' => []]),
    ]);

    $result = rememberValidationBrainService()->recall(
        'brain validation query',
        5,
        ['project' => $project],
        $workspace->id,
    );

    expect($result)->toBe([
        'memories' => [],
        'scores' => [],
    ]);

    Http::assertSent(fn (ClientRequest $request): bool => $request->url() === 'https://qdrant.test/collections/openbrain/points/search'
        && $request->method() === 'POST'
        && $request['filter']['must'] === [
            ['key' => 'workspace_id', 'match' => ['value' => $workspace->id]],
            ['key' => 'project', 'match' => ['value' => $project]],
        ]);
});

test('BrainRememberValidation_search_Bad_rejects_queries_longer_than_2000_bytes', function (): void {
    Http::fake();

    expect(fn () => rememberValidationBrainService()->search(
        str_repeat('q', 2001),
        createWorkspace()->id,
    ))->toThrow(\InvalidArgumentException::class, 'query exceeds maximum length of 2000');

    Http::assertNothingSent();
});

test('BrainRememberValidation_search_Good_falls_back_to_mariadb_when_elasticsearch_fails', function (): void {
    $workspace = createWorkspace();
    $matching = rememberValidationMemory([
        'workspace_id' => $workspace->id,
        'content' => 'Fallback search keeps MariaDB discovery available to self-hosters.',
        'project' => 'agent',
    ]);
    rememberValidationMemory([
        'workspace_id' => $workspace->id,
        'content' => 'Other project memory should not match the scoped fallback search.',
        'project' => 'other-project',
    ]);

    Http::fake([
        'https://elasticsearch.test/brain_memories/_search' => Http::response(['error' => 'unavailable'], 503),
    ]);

    $result = rememberValidationBrainService()->search(
        'Fallback search',
        $workspace->id,
        ['project' => 'agent'],
        5,
    );

    expect($result)->toHaveCount(1)
        ->and($result[0]['id'])->toBe($matching->id)
        ->and($result[0]['score'])->toBe(0.0)
        ->and($result[0]['project'])->toBe('agent');

    Http::assertSent(fn (ClientRequest $request): bool => $request->url() === 'https://elasticsearch.test/brain_memories/_search'
        && $request->method() === 'POST');
});

test('BrainRememberValidation_discoverTags_Bad_rejects_limits_above_100', function (): void {
    expect(fn () => rememberValidationBrainService()->discoverTags(
        createWorkspace()->id,
        limit: 101,
    ))->toThrow(\InvalidArgumentException::class, 'limit must be between 1 and 100');
});

test('BrainRememberValidation_discoverTags_Good_counts_tags_within_scope_and_ignores_blank_tags', function (): void {
    $workspace = createWorkspace();
    rememberValidationMemory([
        'workspace_id' => $workspace->id,
        'tags' => ['openbrain', 'architecture', '  '],
        'project' => 'agent',
    ]);
    rememberValidationMemory([
        'workspace_id' => $workspace->id,
        'tags' => ['openbrain'],
        'project' => 'agent',
    ]);
    rememberValidationMemory([
        'workspace_id' => $workspace->id,
        'tags' => ['deploy'],
        'project' => 'other-project',
    ]);

    $result = rememberValidationBrainService()->discoverTags($workspace->id, 'core', 'agent', 2);

    expect($result)->toBe([
        ['name' => 'openbrain', 'count' => 2],
        ['name' => 'architecture', 'count' => 1],
    ]);
});

test('BrainRememberValidation_listScopes_Ugly_returns_sorted_scope_counts', function (): void {
    $workspace = createWorkspace();
    rememberValidationMemory([
        'workspace_id' => $workspace->id,
        'org' => 'core',
        'project' => 'host',
    ]);
    rememberValidationMemory([
        'workspace_id' => $workspace->id,
        'org' => 'core',
        'project' => 'agent',
    ]);
    rememberValidationMemory([
        'workspace_id' => $workspace->id,
        'org' => null,
        'project' => null,
    ]);
    rememberValidationMemory([
        'workspace_id' => createWorkspace()->id,
        'org' => 'ops',
        'project' => 'deploy',
    ]);

    $result = rememberValidationBrainService()->listScopes($workspace->id);

    expect($result)->toBe([
        [
            'org' => null,
            'count' => 1,
            'projects' => [],
        ],
        [
            'org' => 'core',
            'count' => 2,
            'projects' => [
                ['name' => 'agent', 'count' => 1],
                ['name' => 'host', 'count' => 1],
            ],
        ],
    ]);
});

test('BrainRememberValidation_forget_Bad_rejects_ids_longer_than_64_characters', function (): void {
    expect(fn () => rememberValidationBrainService()->forget(str_repeat('x', 65)))
        ->toThrow(\InvalidArgumentException::class, 'id exceeds maximum length of 64');
});
