<?php

// SPDX-License-Identifier: EUPL-1.2

declare(strict_types=1);

use Core\Mod\Agentic\Console\Commands\BrainCleanCommand;
use Core\Mod\Agentic\Jobs\DeleteFromIndex;
use Core\Mod\Agentic\Models\BrainMemory;
use Illuminate\Console\Command;
use Illuminate\Contracts\Console\Kernel;
use Illuminate\Support\Facades\Queue;

beforeEach(function (): void {
    $this->app->make(Kernel::class)->registerCommand(
        $this->app->make(BrainCleanCommand::class),
    );
});

test('BrainCleanCommand_handle_Good_dispatches_delete_jobs_for_soft_deleted_memories', function (): void {
    Queue::fake();

    $workspace = createWorkspace();
    $deletedMemories = [];

    foreach (range(1, 3) as $index) {
        $memory = BrainMemory::create([
            'workspace_id' => $workspace->id,
            'agent_id' => 'virgil',
            'type' => 'observation',
            'content' => "Soft-deleted memory {$index}.",
            'confidence' => 0.8,
        ]);
        $memory->delete();

        $deletedMemories[] = $memory;
    }

    $activeMemory = BrainMemory::create([
        'workspace_id' => $workspace->id,
        'agent_id' => 'virgil',
        'type' => 'observation',
        'content' => 'Active memories stay indexed.',
        'confidence' => 0.9,
    ]);

    $this->artisan('brain:clean', ['--chunk' => 2])
        ->expectsOutput('Dispatched 3 index cleanup job(s) for soft-deleted brain memories.')
        ->assertSuccessful();

    Queue::assertPushed(DeleteFromIndex::class, 3);

    foreach ($deletedMemories as $memory) {
        Queue::assertPushed(
            DeleteFromIndex::class,
            fn (DeleteFromIndex $job): bool => $job->memoryId === $memory->id,
        );
    }

    Queue::assertNotPushed(
        DeleteFromIndex::class,
        fn (DeleteFromIndex $job): bool => $job->memoryId === $activeMemory->id,
    );
});

test('BrainCleanCommand_handle_Bad_reports_dry_run_without_dispatching_jobs', function (): void {
    Queue::fake();

    $workspace = createWorkspace();
    $memory = BrainMemory::create([
        'workspace_id' => $workspace->id,
        'agent_id' => 'virgil',
        'type' => 'observation',
        'content' => 'Dry-run should only report cleanup work.',
        'confidence' => 0.8,
    ]);
    $memory->delete();

    $this->artisan('brain:clean', ['--dry-run' => true])
        ->expectsOutput('DRY RUN: 1 soft-deleted brain memory record(s) would be removed from indexes.')
        ->assertSuccessful();

    Queue::assertNotPushed(DeleteFromIndex::class);
});

test('BrainCleanCommand_handle_Ugly_rejects_invalid_chunk_size', function (): void {
    Queue::fake();

    $this->artisan('brain:clean', ['--chunk' => 0])
        ->expectsOutput('--chunk must be greater than zero.')
        ->assertExitCode(Command::FAILURE);

    Queue::assertNotPushed(DeleteFromIndex::class);
});
