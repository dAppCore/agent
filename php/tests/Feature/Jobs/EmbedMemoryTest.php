<?php

// SPDX-License-Identifier: EUPL-1.2

declare(strict_types=1);

use Core\Mod\Agentic\Jobs\EmbedMemory;
use Core\Mod\Agentic\Models\BrainMemory;
use Core\Mod\Agentic\Services\BrainService;
use Illuminate\Http\Client\Request;
use Illuminate\Support\Facades\Http;
use Illuminate\Support\Str;

function embedMemoryService(): BrainService
{
    return new BrainService(
        ollamaUrl: 'https://ollama.test',
        qdrantUrl: 'https://qdrant.test',
        collection: 'openbrain',
        embeddingModel: 'embeddinggemma',
        verifySsl: false,
    );
}

test('EmbedMemory_handle_Good_embeds_upserts_and_marks_indexed', function (): void {
    $workspace = createWorkspace();
    $embedding = array_fill(0, 768, 0.125);

    Http::fake([
        'https://ollama.test/api/embeddings' => Http::response(['embedding' => $embedding]),
        'https://qdrant.test/collections/openbrain/points' => Http::response(['result' => ['status' => 'ok']]),
    ]);

    $memory = BrainMemory::create([
        'workspace_id' => $workspace->id,
        'agent_id' => 'virgil',
        'type' => 'architecture',
        'content' => 'Queue memory indexing through EmbedMemory.',
        'tags' => ['brain', 'indexing'],
        'project' => 'agent',
        'confidence' => 0.95,
        'source' => 'ticket-56',
    ]);

    (new EmbedMemory($memory->id))->handle(embedMemoryService());

    expect($memory->fresh()->indexed_at)->not->toBeNull();

    Http::assertSent(fn (Request $request): bool => $request->url() === 'https://ollama.test/api/embeddings'
        && $request->method() === 'POST'
        && $request['model'] === 'embeddinggemma'
        && $request['prompt'] === 'Queue memory indexing through EmbedMemory.');

    Http::assertSent(fn (Request $request): bool => $request->url() === 'https://qdrant.test/collections/openbrain/points'
        && $request->method() === 'PUT'
        && $request['points'][0]['id'] === $memory->id
        && $request['points'][0]['vector'] === $embedding
        && $request['points'][0]['payload']['workspace_id'] === $workspace->id
        && $request['points'][0]['payload']['project'] === 'agent'
        && $request['points'][0]['payload']['agent_id'] === 'virgil'
        && $request['points'][0]['payload']['type'] === 'architecture'
        && $request['points'][0]['payload']['tags'] === ['brain', 'indexing']
        && $request['points'][0]['payload']['confidence'] === 0.95
        && $request['points'][0]['payload']['source'] === 'ticket-56'
        && $request['points'][0]['payload']['content'] === 'Queue memory indexing through EmbedMemory.');
});

test('EmbedMemory_handle_Bad_returns_silently_when_memory_is_missing', function (): void {
    Http::fake();

    (new EmbedMemory(Str::uuid()->toString()))->handle(embedMemoryService());

    Http::assertNothingSent();
});

test('EmbedMemory_handle_Ugly_leaves_memory_unindexed_when_qdrant_fails', function (): void {
    $workspace = createWorkspace();

    Http::fake([
        'https://ollama.test/api/embeddings' => Http::response(['embedding' => array_fill(0, 768, 0.25)]),
        'https://qdrant.test/collections/openbrain/points' => Http::response(['error' => 'unavailable'], 500),
    ]);

    $memory = BrainMemory::create([
        'workspace_id' => $workspace->id,
        'agent_id' => 'virgil',
        'type' => 'observation',
        'content' => 'Qdrant failures should retry without marking the memory indexed.',
        'confidence' => 0.8,
    ]);

    try {
        (new EmbedMemory($memory->id))->handle(embedMemoryService());
        $this->fail('Expected Qdrant failure to throw.');
    } catch (RuntimeException $exception) {
        expect($exception->getMessage())->toBe('Qdrant upsert failed: 500');
    }

    expect($memory->fresh()->indexed_at)->toBeNull();
});
