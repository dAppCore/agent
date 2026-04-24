<?php

// SPDX-License-Identifier: EUPL-1.2

declare(strict_types=1);

use Core\Mod\Agentic\Jobs\DeleteFromIndex;
use Core\Mod\Agentic\Jobs\EmbedMemory;
use Core\Mod\Agentic\Models\BrainMemory;
use Core\Mod\Agentic\Services\BrainService;
use Illuminate\Support\Facades\Queue;

function cleanupBrainService(): BrainService
{
    return new BrainService;
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

test('SupersedeForgetIndexCleanup_forget_Good_dispatches_delete_from_index', function (): void {
    Queue::fake();
    $memory = cleanupMemory(['indexed_at' => now()]);

    cleanupBrainService()->forget($memory->id);

    expect(BrainMemory::find($memory->id))->toBeNull()
        ->and(BrainMemory::withTrashed()->find($memory->id)?->trashed())->toBeTrue();

    Queue::assertPushed(DeleteFromIndex::class, fn (DeleteFromIndex $job): bool => $job->memoryId === $memory->id);
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

test('SupersedeForgetIndexCleanup_supersede_Ugly_skips_cleanup_for_never_indexed_memory', function (): void {
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

    Queue::assertNotPushed(DeleteFromIndex::class, fn (DeleteFromIndex $job): bool => $job->memoryId === $oldMemory->id);
    Queue::assertPushed(EmbedMemory::class, fn (EmbedMemory $job): bool => $job->memoryId === $newMemory->id);
});
