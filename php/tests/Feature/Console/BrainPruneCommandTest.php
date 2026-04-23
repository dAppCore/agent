<?php

// SPDX-License-Identifier: EUPL-1.2

declare(strict_types=1);

use Core\Mod\Agentic\Console\Commands\BrainPruneCommand;
use Core\Mod\Agentic\Jobs\DeleteFromIndex;
use Core\Mod\Agentic\Models\BrainMemory;
use Illuminate\Console\Command;
use Illuminate\Contracts\Console\Kernel;
use Illuminate\Support\Carbon;
use Illuminate\Support\Facades\Queue;

beforeEach(function (): void {
    Carbon::setTestNow(Carbon::parse('2026-04-23 12:00:00'));

    $this->app->make(Kernel::class)->registerCommand(
        $this->app->make(BrainPruneCommand::class),
    );
});

afterEach(function (): void {
    Carbon::setTestNow();
});

function brainPruneMemory(array $attributes = []): BrainMemory
{
    return BrainMemory::create(array_merge([
        'workspace_id' => $attributes['workspace_id'] ?? createWorkspace()->id,
        'agent_id' => 'virgil',
        'type' => 'observation',
        'content' => 'Brain prune command test memory.',
        'confidence' => 0.8,
    ], $attributes));
}

function brainPruneSoftDelete(BrainMemory $memory, int $daysAgo): BrainMemory
{
    $memory->delete();
    $memory->forceFill([
        'deleted_at' => now()->subDays($daysAgo),
    ])->save();

    return BrainMemory::onlyTrashed()->findOrFail($memory->id);
}

test('BrainPruneCommand_handle_Good_force_deletes_stale_soft_deleted_memories', function (): void {
    Queue::fake();

    $workspace = createWorkspace();
    $firstStaleMemory = brainPruneSoftDelete(
        brainPruneMemory([
            'workspace_id' => $workspace->id,
            'content' => 'First stale memory.',
        ]),
        91,
    );
    $secondStaleMemory = brainPruneSoftDelete(
        brainPruneMemory([
            'workspace_id' => $workspace->id,
            'content' => 'Second stale memory.',
        ]),
        120,
    );
    $recentMemory = brainPruneSoftDelete(
        brainPruneMemory([
            'workspace_id' => $workspace->id,
            'content' => 'Recently deleted memory.',
        ]),
        90,
    );
    $activeMemory = brainPruneMemory([
        'workspace_id' => $workspace->id,
        'content' => 'Active memory.',
    ]);

    $this->artisan('brain:prune', ['--older-than' => 90, '--chunk' => 1])
        ->expectsOutput('Pruned 2 stale soft-deleted brain memory record(s).')
        ->assertSuccessful();

    Queue::assertPushed(DeleteFromIndex::class, 2);
    Queue::assertPushed(DeleteFromIndex::class, fn (DeleteFromIndex $job): bool => $job->memoryId === $firstStaleMemory->id);
    Queue::assertPushed(DeleteFromIndex::class, fn (DeleteFromIndex $job): bool => $job->memoryId === $secondStaleMemory->id);
    Queue::assertNotPushed(DeleteFromIndex::class, fn (DeleteFromIndex $job): bool => $job->memoryId === $recentMemory->id);
    Queue::assertNotPushed(DeleteFromIndex::class, fn (DeleteFromIndex $job): bool => $job->memoryId === $activeMemory->id);

    expect(BrainMemory::withTrashed()->find($firstStaleMemory->id))->toBeNull()
        ->and(BrainMemory::withTrashed()->find($secondStaleMemory->id))->toBeNull()
        ->and(BrainMemory::onlyTrashed()->find($recentMemory->id))->not->toBeNull()
        ->and(BrainMemory::query()->find($activeMemory->id))->not->toBeNull();
});

test('BrainPruneCommand_handle_Bad_reports_dry_run_without_dispatching_jobs_or_deleting_records', function (): void {
    Queue::fake();

    $memory = brainPruneSoftDelete(brainPruneMemory(), 180);

    $this->artisan('brain:prune', ['--dry-run' => true])
        ->expectsOutput('DRY RUN: 1 stale soft-deleted brain memory record(s) would be permanently deleted.')
        ->assertSuccessful();

    Queue::assertNotPushed(DeleteFromIndex::class);

    expect(BrainMemory::onlyTrashed()->find($memory->id))->not->toBeNull();
});

test('BrainPruneCommand_handle_Ugly_rejects_invalid_retention_window', function (): void {
    Queue::fake();

    brainPruneSoftDelete(brainPruneMemory(), 180);

    $this->artisan('brain:prune', ['--older-than' => 0])
        ->expectsOutput('--older-than must be a positive integer.')
        ->assertExitCode(Command::FAILURE);

    Queue::assertNotPushed(DeleteFromIndex::class);
});
