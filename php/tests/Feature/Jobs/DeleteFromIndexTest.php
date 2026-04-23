<?php

// SPDX-License-Identifier: EUPL-1.2

declare(strict_types=1);

use Core\Mod\Agentic\Jobs\DeleteFromIndex;
use Core\Mod\Agentic\Models\BrainMemory;
use Core\Mod\Agentic\Services\BrainService;
use Illuminate\Http\Client\Request;
use Illuminate\Support\Facades\Http;
use Illuminate\Support\Str;

function deleteFromIndexService(): BrainService
{
    return new BrainService(
        ollamaUrl: 'https://ollama.test',
        qdrantUrl: 'https://qdrant.test',
        collection: 'openbrain',
        embeddingModel: 'embeddinggemma',
        verifySsl: false,
    );
}

test('DeleteFromIndex_handle_Good_deletes_memory_from_qdrant', function (): void {
    $memoryId = Str::uuid()->toString();

    Http::fake([
        'https://qdrant.test/collections/openbrain/points/delete' => Http::response(['result' => ['status' => 'ok']]),
    ]);

    (new DeleteFromIndex($memoryId))->handle(deleteFromIndexService());

    Http::assertSent(fn (Request $request): bool => $request->url() === 'https://qdrant.test/collections/openbrain/points/delete'
        && $request->method() === 'POST'
        && $request['points'] === [$memoryId]);
});

test('DeleteFromIndex_handle_Bad_deletes_soft_deleted_memory_from_qdrant', function (): void {
    $workspace = createWorkspace();

    Http::fake([
        'https://qdrant.test/collections/openbrain/points/delete' => Http::response(['result' => ['status' => 'ok']]),
    ]);

    $memory = BrainMemory::create([
        'workspace_id' => $workspace->id,
        'agent_id' => 'virgil',
        'type' => 'observation',
        'content' => 'Soft-deleted memories should still be removed from the index.',
        'confidence' => 0.8,
    ]);
    $memory->delete();

    (new DeleteFromIndex($memory->id))->handle(deleteFromIndexService());

    Http::assertSent(fn (Request $request): bool => $request->url() === 'https://qdrant.test/collections/openbrain/points/delete'
        && $request->method() === 'POST'
        && $request['points'] === [$memory->id]);
});

test('DeleteFromIndex_handle_Ugly_throws_when_qdrant_delete_fails', function (): void {
    $memoryId = Str::uuid()->toString();

    Http::fake([
        'https://qdrant.test/collections/openbrain/points/delete' => Http::response(['error' => 'unavailable'], 500),
    ]);

    try {
        (new DeleteFromIndex($memoryId))->handle(deleteFromIndexService());
        $this->fail('Expected Qdrant failure to throw.');
    } catch (RuntimeException $exception) {
        expect($exception->getMessage())->toBe('Qdrant delete failed: 500');
    }
});
