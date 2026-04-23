<?php

// SPDX-License-Identifier: EUPL-1.2

declare(strict_types=1);

use Core\Mod\Agentic\Console\Commands\BrainReindexCommand;
use Core\Mod\Agentic\Jobs\EmbedMemory;
use Core\Mod\Agentic\Models\BrainMemory;
use Illuminate\Contracts\Console\Kernel;
use Illuminate\Support\Facades\Queue;

beforeEach(function (): void {
    $this->app->make(Kernel::class)->registerCommand(new BrainReindexCommand());
});

function brainReindexMemory(array $attributes = []): BrainMemory
{
    return BrainMemory::create(array_merge([
        'workspace_id' => $attributes['workspace_id'] ?? createWorkspace()->id,
        'agent_id' => 'virgil',
        'type' => 'observation',
        'content' => 'Brain reindex command test memory.',
        'confidence' => 0.9,
    ], $attributes));
}

test('BrainReindexCommand_handle_Good_dispatches_unindexed_memories', function (): void {
    Queue::fake();
    $workspace = createWorkspace();
    $firstMemory = brainReindexMemory(['workspace_id' => $workspace->id]);
    $secondMemory = brainReindexMemory([
        'workspace_id' => $workspace->id,
        'content' => 'Second unindexed memory.',
    ]);
    brainReindexMemory([
        'workspace_id' => $workspace->id,
        'content' => 'Already indexed memory.',
        'indexed_at' => now(),
    ]);

    $this->artisan('brain:reindex', ['--chunk' => 1])
        ->expectsOutputToContain('Dispatched 2 brain memory embedding job(s) for unindexed memories.')
        ->assertSuccessful();

    Queue::assertPushed(EmbedMemory::class, 2);
    Queue::assertPushed(EmbedMemory::class, fn (EmbedMemory $job): bool => $job->memoryId === $firstMemory->id);
    Queue::assertPushed(EmbedMemory::class, fn (EmbedMemory $job): bool => $job->memoryId === $secondMemory->id);
});

test('BrainReindexCommand_handle_Bad_rejects_invalid_chunk_size', function (): void {
    Queue::fake();
    brainReindexMemory();

    $this->artisan('brain:reindex', ['--chunk' => 0])
        ->expectsOutputToContain('--chunk must be a positive integer.')
        ->assertFailed();

    Queue::assertNothingPushed();
});

test('BrainReindexCommand_handle_Ugly_all_reindexes_every_memory_across_chunks', function (): void {
    Queue::fake();
    $workspace = createWorkspace();
    $unindexedMemory = brainReindexMemory(['workspace_id' => $workspace->id]);
    $indexedMemory = brainReindexMemory([
        'workspace_id' => $workspace->id,
        'content' => 'Previously indexed memory.',
        'indexed_at' => now(),
    ]);

    $this->artisan('brain:reindex', [
        '--all' => true,
        '--chunk' => 1,
    ])
        ->expectsOutputToContain('Dispatched 2 brain memory embedding job(s) for all memories.')
        ->assertSuccessful();

    Queue::assertPushed(EmbedMemory::class, 2);
    Queue::assertPushed(EmbedMemory::class, fn (EmbedMemory $job): bool => $job->memoryId === $unindexedMemory->id);
    Queue::assertPushed(EmbedMemory::class, fn (EmbedMemory $job): bool => $job->memoryId === $indexedMemory->id);
});
