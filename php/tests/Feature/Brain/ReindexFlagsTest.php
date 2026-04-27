<?php

// SPDX-License-Identifier: EUPL-1.2

declare(strict_types=1);

use Core\Mod\Agentic\Console\Commands\BrainReindexCommand;
use Core\Mod\Agentic\Jobs\EmbedMemory;
use Core\Mod\Agentic\Models\BrainMemory;
use Illuminate\Contracts\Console\Kernel;
use Illuminate\Queue\CallQueuedClosure;
use Illuminate\Support\Facades\Queue;

beforeEach(function (): void {
    $this->app->make(Kernel::class)->registerCommand(new BrainReindexCommand());
});

function reindexFlagsMemory(array $attributes = []): BrainMemory
{
    return BrainMemory::create(array_merge([
        'workspace_id' => $attributes['workspace_id'] ?? createWorkspace()->id,
        'agent_id' => 'virgil',
        'type' => 'observation',
        'content' => 'Brain reindex flags test memory.',
        'confidence' => 0.83,
        'org' => 'core',
        'project' => 'agent',
    ], $attributes));
}

test('BrainReindexCommand_handle_Good_filters_by_org_and_project', function (): void {
    Queue::fake();
    $workspace = createWorkspace();
    $matching = reindexFlagsMemory([
        'workspace_id' => $workspace->id,
        'org' => 'core',
        'project' => 'agent',
    ]);
    reindexFlagsMemory([
        'workspace_id' => $workspace->id,
        'org' => 'core',
        'project' => 'other-project',
    ]);
    reindexFlagsMemory([
        'workspace_id' => $workspace->id,
        'org' => 'other-org',
        'project' => 'agent',
    ]);

    $this->artisan('brain:reindex', [
        '--org' => 'core',
        '--project' => 'agent',
        '--chunk' => 1,
    ])
        ->expectsOutputToContain('Dispatched 1 brain memory embedding job(s) for unindexed memories.')
        ->assertSuccessful();

    Queue::assertPushed(EmbedMemory::class, 1);
    Queue::assertPushed(EmbedMemory::class, fn (EmbedMemory $job): bool => $job->memoryId === $matching->id);
});

test('BrainReindexCommand_handle_Bad_filters_stale_memories', function (): void {
    Queue::fake();
    $workspace = createWorkspace();
    $neverIndexed = reindexFlagsMemory([
        'workspace_id' => $workspace->id,
        'content' => 'Never indexed memory.',
        'indexed_at' => null,
    ]);
    $stale = reindexFlagsMemory([
        'workspace_id' => $workspace->id,
        'content' => 'Stale indexed memory.',
        'indexed_at' => now()->subDays(21),
    ]);
    $recent = reindexFlagsMemory([
        'workspace_id' => $workspace->id,
        'content' => 'Recently indexed memory.',
        'indexed_at' => now()->subDays(3),
    ]);

    $this->artisan('brain:reindex', [
        '--stale' => true,
        '--chunk' => 1,
    ])
        ->expectsOutputToContain('Dispatched 2 brain memory embedding job(s) for stale memories.')
        ->assertSuccessful();

    Queue::assertPushed(EmbedMemory::class, 2);
    Queue::assertPushed(EmbedMemory::class, fn (EmbedMemory $job): bool => $job->memoryId === $neverIndexed->id);
    Queue::assertPushed(EmbedMemory::class, fn (EmbedMemory $job): bool => $job->memoryId === $stale->id);
    Queue::assertNotPushed(EmbedMemory::class, fn (EmbedMemory $job): bool => $job->memoryId === $recent->id);
});

test('BrainReindexCommand_handle_Ugly_dry_run_counts_matches_without_queueing_jobs', function (): void {
    Queue::fake();
    $workspace = createWorkspace();
    reindexFlagsMemory([
        'workspace_id' => $workspace->id,
        'org' => 'core',
    ]);
    reindexFlagsMemory([
        'workspace_id' => $workspace->id,
        'org' => 'other-org',
    ]);

    $this->artisan('brain:reindex', [
        '--org' => 'core',
        '--dry-run' => true,
        '--chunk' => 1,
    ])
        ->expectsOutputToContain('DRY RUN: 1 brain memory record(s) match unindexed reindex filters.')
        ->assertSuccessful();

    Queue::assertNothingPushed();
});

test('BrainReindexCommand_handle_Good_elastic_only_dispatches_lighter_reindex_jobs', function (): void {
    Queue::fake();
    $workspace = createWorkspace();
    reindexFlagsMemory([
        'workspace_id' => $workspace->id,
        'indexed_at' => now()->subDays(30),
        'org' => 'core',
    ]);

    $this->artisan('brain:reindex', [
        '--all' => true,
        '--org' => 'core',
        '--elastic-only' => true,
        '--chunk' => 1,
    ])
        ->expectsOutputToContain('Dispatched 1 brain memory elastic-only reindex job(s) for all memories.')
        ->assertSuccessful();

    Queue::assertNotPushed(EmbedMemory::class);
    Queue::assertPushed(CallQueuedClosure::class, 1);
});
