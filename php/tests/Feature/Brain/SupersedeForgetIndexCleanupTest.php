<?php

// SPDX-License-Identifier: EUPL-1.2

declare(strict_types=1);

use Core\Mod\Agentic\Jobs\DeleteFromIndex;
use Core\Mod\Agentic\Jobs\EmbedMemory;
use Core\Mod\Agentic\Models\BrainMemory;
use Core\Mod\Agentic\Services\BrainService;
use Illuminate\Support\Str;
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
        ->and($newMemory->indexed_at)->toBeNull()
        ->and($newMemory->supersedes_id)->toBe($oldMemory->id);

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
        ->and($newMemory->indexed_at)->toBeNull()
        ->and($newMemory->supersedes_id)->toBe($oldMemory->id);

    Queue::assertPushed(DeleteFromIndex::class, fn (DeleteFromIndex $job): bool => $job->memoryId === $oldMemory->id);
    Queue::assertPushed(EmbedMemory::class, fn (EmbedMemory $job): bool => $job->memoryId === $newMemory->id);
});

test('SupersedeForgetIndexCleanup_supersede_Good_retries_follow_the_current_head', function (): void {
    Queue::fake();
    $brain = cleanupBrainService();
    $workspace = createWorkspace();
    $originalMemory = cleanupMemory([
        'workspace_id' => $workspace->id,
        'content' => 'Original memory for retry chain.',
    ]);

    $replacement = $brain->remember([
        'workspace_id' => $workspace->id,
        'agent_id' => 'virgil',
        'type' => 'observation',
        'content' => 'First replacement memory.',
        'confidence' => 0.91,
        'org' => 'core',
        'project' => 'agent',
        'supersedes_id' => $originalMemory->id,
    ]);

    $retriedReplacement = $brain->remember([
        'workspace_id' => $workspace->id,
        'agent_id' => 'virgil',
        'type' => 'observation',
        'content' => 'Retried replacement memory.',
        'confidence' => 0.92,
        'org' => 'core',
        'project' => 'agent',
        'supersedes_id' => $originalMemory->id,
    ]);

    expect($replacement->supersedes_id)->toBe($originalMemory->id)
        ->and($retriedReplacement->supersedes_id)->toBe($replacement->id)
        ->and(BrainMemory::find($originalMemory->id))->toBeNull()
        ->and(BrainMemory::withTrashed()->find($originalMemory->id)?->trashed())->toBeTrue()
        ->and(BrainMemory::find($replacement->id))->toBeNull()
        ->and(BrainMemory::withTrashed()->find($replacement->id)?->trashed())->toBeTrue()
        ->and(BrainMemory::find($retriedReplacement->id))->not->toBeNull();

    Queue::assertPushed(DeleteFromIndex::class, 2);
    Queue::assertPushed(DeleteFromIndex::class, fn (DeleteFromIndex $job): bool => $job->memoryId === $originalMemory->id);
    Queue::assertPushed(DeleteFromIndex::class, fn (DeleteFromIndex $job): bool => $job->memoryId === $replacement->id);
    Queue::assertPushed(EmbedMemory::class, 2);
});

test('SupersedeForgetIndexCleanup_supersede_Bad_throws_for_unknown_memory', function (): void {
    Queue::fake();
    $workspace = createWorkspace();
    $missingMemoryId = Str::uuid()->toString();

    expect(fn () => cleanupBrainService()->remember([
        'workspace_id' => $workspace->id,
        'agent_id' => 'virgil',
        'type' => 'observation',
        'content' => 'Memory with missing supersede target.',
        'confidence' => 0.88,
        'org' => 'core',
        'project' => 'agent',
        'supersedes_id' => $missingMemoryId,
    ]))->toThrow(\InvalidArgumentException::class, "Superseded memory not found: {$missingMemoryId}");

    expect(BrainMemory::count())->toBe(0);
    Queue::assertNothingPushed();
});

test('SupersedeForgetIndexCleanup_supersede_Ugly_detects_cycles_before_writing', function (): void {
    Queue::fake();
    $workspace = createWorkspace();
    $firstMemory = cleanupMemory([
        'workspace_id' => $workspace->id,
        'content' => 'Cycle root memory.',
    ]);
    $secondMemory = cleanupMemory([
        'workspace_id' => $workspace->id,
        'content' => 'Cycle successor memory.',
        'supersedes_id' => $firstMemory->id,
    ]);

    $firstMemory->supersedes_id = $secondMemory->id;
    $firstMemory->save();

    expect(fn () => cleanupBrainService()->remember([
        'workspace_id' => $workspace->id,
        'agent_id' => 'virgil',
        'type' => 'observation',
        'content' => 'Memory that should fail on cycle.',
        'confidence' => 0.9,
        'org' => 'core',
        'project' => 'agent',
        'supersedes_id' => $firstMemory->id,
    ]))->toThrow(\RuntimeException::class, "Detected cycle while resolving supersede chain: {$firstMemory->id}");

    expect(BrainMemory::count())->toBe(2)
        ->and(BrainMemory::withTrashed()->find($firstMemory->id)?->trashed())->toBeFalse()
        ->and(BrainMemory::withTrashed()->find($secondMemory->id)?->trashed())->toBeFalse();
    Queue::assertNothingPushed();
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
