<?php

// SPDX-License-Identifier: EUPL-1.2

declare(strict_types=1);

use Core\Mod\Agentic\Jobs\DeleteFromIndex;
use Core\Mod\Agentic\Jobs\EmbedMemory;
use Core\Mod\Agentic\Models\BrainMemory;
use Core\Mod\Agentic\Services\BrainService;
use Illuminate\Support\Facades\Http;
use Illuminate\Support\Facades\Queue;

function cleanupBrainService(): BrainService
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

function cleanupMemory(array $attributes = []): BrainMemory
{
    return BrainMemory::create(array_merge([
        'workspace_id' => $attributes['workspace_id'] ?? createWorkspace()->id,
        'agent_id' => 'virgil',
        'type' => 'observation',
        'content' => 'Brain cleanup test memory.',
        'confidence' => 0.84,
        'org' => 'core',
        'project' => 'agent',
    ], $attributes));
}

function cleanupQdrantPoint(BrainService $brain, BrainMemory $memory): array
{
    $point = $brain->buildQdrantPayload($memory->id, [
        'workspace_id' => $memory->workspace_id,
        'org' => $memory->getAttribute('org'),
        'project' => $memory->project,
        'agent_id' => $memory->agent_id,
        'type' => $memory->type,
        'tags' => $memory->tags ?? [],
        'confidence' => $memory->confidence,
        'source' => $memory->source ?? 'manual',
        'content' => $memory->content,
        'created_at' => $memory->created_at?->toIso8601String(),
    ]);
    $point['vector'] = array_fill(0, 768, 0.125);

    return $point;
}

test('SupersedeForgetIndexCleanup_forget_Good_dispatches_delete_from_index', function (): void {
    Queue::fake();
    $memory = cleanupMemory(['indexed_at' => now()]);

    cleanupBrainService()->forget($memory->id);

    expect(BrainMemory::find($memory->id))->toBeNull()
        ->and(BrainMemory::withTrashed()->find($memory->id)?->trashed())->toBeTrue();

    Queue::assertPushed(DeleteFromIndex::class, fn (DeleteFromIndex $job): bool => $job->memoryId === $memory->id);
});

test('SupersedeForgetIndexCleanup_forget_Bad_dispatches_delete_from_index_for_never_indexed_memory', function (): void {
    Queue::fake();
    $memory = cleanupMemory(['indexed_at' => null]);

    cleanupBrainService()->forget($memory->id);

    expect(BrainMemory::find($memory->id))->toBeNull()
        ->and(BrainMemory::withTrashed()->find($memory->id)?->trashed())->toBeTrue();

    Queue::assertPushed(DeleteFromIndex::class, fn (DeleteFromIndex $job): bool => $job->memoryId === $memory->id);
});

test('SupersedeForgetIndexCleanup_forget_Ugly_skips_late_index_writes_for_deleted_memory', function (): void {
    Queue::fake();
    Http::fake();
    $brain = cleanupBrainService();
    $memory = cleanupMemory(['indexed_at' => null]);
    $staleMemory = BrainMemory::query()->findOrFail($memory->id);

    $brain->forget($memory->id);
    $brain->qdrantUpsert([cleanupQdrantPoint($brain, $staleMemory)]);
    $brain->elasticIndex($staleMemory);

    expect(BrainMemory::count())->toBe(0)
        ->and(BrainMemory::withTrashed()->find($memory->id)?->trashed())->toBeTrue();

    Queue::assertPushed(DeleteFromIndex::class, 1);
    Queue::assertPushed(DeleteFromIndex::class, fn (DeleteFromIndex $job): bool => $job->memoryId === $memory->id);
    Http::assertNothingSent();
});

test('SupersedeForgetIndexCleanup_supersede_Bad_dispatches_cleanup_for_old_indexed_memory', function (): void {
    Queue::fake();
    $workspace = createWorkspace();
    $oldMemory = cleanupMemory([
        'workspace_id' => $workspace->id,
        'content' => 'Old indexed memory.',
        'indexed_at' => now(),
    ]);

    $newMemory = cleanupBrainService()->remember([
        'workspace_id' => $workspace->id,
        'agent_id' => 'virgil',
        'type' => 'observation',
        'content' => 'New superseding memory.',
        'confidence' => 0.93,
        'org' => 'core',
        'project' => 'agent',
        'supersedes_id' => $oldMemory->id,
    ]);

    expect(BrainMemory::find($oldMemory->id))->toBeNull()
        ->and(BrainMemory::withTrashed()->find($oldMemory->id)?->trashed())->toBeTrue()
        ->and($newMemory->indexed_at)->toBeNull();

    Queue::assertPushed(DeleteFromIndex::class, fn (DeleteFromIndex $job): bool => $job->memoryId === $oldMemory->id);
    Queue::assertPushed(EmbedMemory::class, fn (EmbedMemory $job): bool => $job->memoryId === $newMemory->id);
});

test('SupersedeForgetIndexCleanup_supersede_Ugly_dispatches_cleanup_for_never_indexed_memory', function (): void {
    Queue::fake();
    $workspace = createWorkspace();
    $oldMemory = cleanupMemory([
        'workspace_id' => $workspace->id,
        'content' => 'Old unindexed memory.',
        'indexed_at' => null,
    ]);

    $newMemory = cleanupBrainService()->remember([
        'workspace_id' => $workspace->id,
        'agent_id' => 'virgil',
        'type' => 'observation',
        'content' => 'Superseding unindexed memory.',
        'confidence' => 0.9,
        'org' => 'core',
        'project' => 'agent',
        'supersedes_id' => $oldMemory->id,
    ]);

    expect(BrainMemory::find($oldMemory->id))->toBeNull()
        ->and(BrainMemory::withTrashed()->find($oldMemory->id)?->trashed())->toBeTrue()
        ->and($newMemory->indexed_at)->toBeNull();

    Queue::assertPushed(DeleteFromIndex::class, fn (DeleteFromIndex $job): bool => $job->memoryId === $oldMemory->id);
    Queue::assertPushed(EmbedMemory::class, fn (EmbedMemory $job): bool => $job->memoryId === $newMemory->id);
});

test('SupersedeForgetIndexCleanup_supersede_Ugly_skips_late_index_writes_for_deleted_memory', function (): void {
    Queue::fake();
    Http::fake();
    $brain = cleanupBrainService();
    $workspace = createWorkspace();
    $oldMemory = cleanupMemory([
        'workspace_id' => $workspace->id,
        'content' => 'Old unindexed memory for late write test.',
        'indexed_at' => null,
    ]);
    $staleOldMemory = BrainMemory::query()->findOrFail($oldMemory->id);

    $newMemory = $brain->remember([
        'workspace_id' => $workspace->id,
        'agent_id' => 'virgil',
        'type' => 'observation',
        'content' => 'Superseding memory for late write test.',
        'confidence' => 0.9,
        'org' => 'core',
        'project' => 'agent',
        'supersedes_id' => $oldMemory->id,
    ]);

    $brain->qdrantUpsert([cleanupQdrantPoint($brain, $staleOldMemory)]);
    $brain->elasticIndex($staleOldMemory);

    expect(BrainMemory::count())->toBe(1)
        ->and(BrainMemory::find($oldMemory->id))->toBeNull()
        ->and(BrainMemory::withTrashed()->find($oldMemory->id)?->trashed())->toBeTrue()
        ->and(BrainMemory::find($newMemory->id))->not->toBeNull();

    Queue::assertPushed(DeleteFromIndex::class, fn (DeleteFromIndex $job): bool => $job->memoryId === $oldMemory->id);
    Queue::assertPushed(EmbedMemory::class, fn (EmbedMemory $job): bool => $job->memoryId === $newMemory->id);
    Http::assertNothingSent();
});
