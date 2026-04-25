<?php

// SPDX-License-Identifier: EUPL-1.2

declare(strict_types=1);

use Core\Mod\Agentic\Jobs\EmbedMemory;
use Core\Mod\Agentic\Models\BrainMemory;
use Core\Mod\Agentic\Services\BrainService;
use Illuminate\Support\Facades\Queue;

function rememberBrainService(): BrainService
{
    return new class extends BrainService
    {
        public bool $qdrantUpsertCalled = false;

        /**
         * @return array<float>
         */
        public function embed(string $text): array
        {
            return array_fill(0, 768, 0.125);
        }

        public function qdrantUpsert(array $points): void
        {
            $this->qdrantUpsertCalled = true;
        }
    };
}

function rememberBrainAttributes(array $attributes = []): array
{
    return array_merge([
        'workspace_id' => $attributes['workspace_id'] ?? createWorkspace()->id,
        'agent_id' => 'virgil',
        'type' => 'architecture',
        'content' => 'Brain memories are indexed by queued jobs.',
        'tags' => ['brain', 'queue'],
        'project' => 'agent',
        'confidence' => 0.95,
        'source' => 'ticket-55',
    ], $attributes);
}

test('BrainService_remember_Good_returns_unindexed_memory_and_dispatches_embed_job', function (): void {
    Queue::fake();
    $brain = rememberBrainService();

    $memory = $brain->remember(rememberBrainAttributes([
        'indexed_at' => now(),
    ]));

    expect($memory->exists)->toBeTrue()
        ->and($memory->indexed_at)->toBeNull()
        ->and($memory->fresh()->indexed_at)->toBeNull()
        ->and($brain->qdrantUpsertCalled)->toBeFalse();

    Queue::assertPushed(EmbedMemory::class, fn (EmbedMemory $job): bool => $job->memoryId === $memory->id);
});

test('BrainService_remember_Bad_does_not_call_qdrant_upsert_directly', function (): void {
    Queue::fake();
    $brain = rememberBrainService();

    $brain->remember(rememberBrainAttributes());

    expect($brain->qdrantUpsertCalled)->toBeFalse();

    Queue::assertPushed(EmbedMemory::class);
});

test('BrainService_remember_Ugly_soft_deletes_superseded_memory_before_dispatching_job', function (): void {
    Queue::fake();
    $brain = rememberBrainService();
    $workspace = createWorkspace();
    $oldMemory = BrainMemory::create(rememberBrainAttributes([
        'workspace_id' => $workspace->id,
        'content' => 'Old memory version.',
    ]));

    $memory = $brain->remember(rememberBrainAttributes([
        'workspace_id' => $workspace->id,
        'content' => 'New memory version.',
        'supersedes_id' => $oldMemory->id,
    ]));

    expect(BrainMemory::find($oldMemory->id))->toBeNull()
        ->and(BrainMemory::withTrashed()->find($oldMemory->id)?->trashed())->toBeTrue()
        ->and($memory->indexed_at)->toBeNull()
        ->and($brain->qdrantUpsertCalled)->toBeFalse();

    Queue::assertPushed(EmbedMemory::class, fn (EmbedMemory $job): bool => $job->memoryId === $memory->id);
});
