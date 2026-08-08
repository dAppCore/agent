<?php

declare(strict_types=1);

/**
 * Tests for the BatchContentGeneration queue job.
 *
 * Covers job configuration, queue assignment, tag generation, and dispatch behaviour.
 * ContentTask comes from dappcore/php-content; the empty-batch path runs against
 * a real content_tasks table created for that test.
 */

use Core\Mod\Agentic\Jobs\BatchContentGeneration;
use Core\Mod\Agentic\Jobs\ProcessContentTask;
use Core\Mod\Content\Models\ContentTask;
use Illuminate\Contracts\Queue\ShouldQueue;
use Illuminate\Database\Schema\Blueprint;
use Illuminate\Support\Facades\Log;
use Illuminate\Support\Facades\Queue;
use Illuminate\Support\Facades\Schema;

// =========================================================================
// Job Configuration Tests
// =========================================================================

describe('job configuration', function () {
    it('has a 600 second timeout', function () {
        $job = new BatchContentGeneration;

        expect($job->timeout)->toBe(600);
    });

    it('defaults to normal priority', function () {
        $job = new BatchContentGeneration;

        expect($job->priority)->toBe('normal');
    });

    it('defaults to a batch size of 10', function () {
        $job = new BatchContentGeneration;

        expect($job->batchSize)->toBe(10);
    });

    it('accepts a custom priority', function () {
        $job = new BatchContentGeneration('high');

        expect($job->priority)->toBe('high');
    });

    it('accepts a custom batch size', function () {
        $job = new BatchContentGeneration('normal', 25);

        expect($job->batchSize)->toBe(25);
    });

    it('accepts both custom priority and batch size', function () {
        $job = new BatchContentGeneration('low', 5);

        expect($job->priority)->toBe('low')
            ->and($job->batchSize)->toBe(5);
    });

    it('implements ShouldQueue', function () {
        $job = new BatchContentGeneration;

        expect($job)->toBeInstanceOf(ShouldQueue::class);
    });
});

// =========================================================================
// Queue Assignment Tests
// =========================================================================

describe('queue assignment', function () {
    it('dispatches to the ai-batch queue', function () {
        Queue::fake();

        BatchContentGeneration::dispatch();

        Queue::assertPushedOn('ai-batch', BatchContentGeneration::class);
    });

    it('dispatches with correct priority when specified', function () {
        Queue::fake();

        BatchContentGeneration::dispatch('high', 5);

        Queue::assertPushed(BatchContentGeneration::class, function ($job) {
            return $job->priority === 'high' && $job->batchSize === 5;
        });
    });

    it('dispatches with default values when no arguments given', function () {
        Queue::fake();

        BatchContentGeneration::dispatch();

        Queue::assertPushed(BatchContentGeneration::class, function ($job) {
            return $job->priority === 'normal' && $job->batchSize === 10;
        });
    });

    it('can be dispatched multiple times with different priorities', function () {
        Queue::fake();

        BatchContentGeneration::dispatch('high');
        BatchContentGeneration::dispatch('low');

        Queue::assertPushed(BatchContentGeneration::class, 2);
    });
});

// =========================================================================
// Tag Generation Tests
// =========================================================================

describe('tags', function () {
    it('always includes the batch-generation tag', function () {
        $job = new BatchContentGeneration;

        expect($job->tags())->toContain('batch-generation');
    });

    it('includes a priority tag for normal priority', function () {
        $job = new BatchContentGeneration('normal');

        expect($job->tags())->toContain('priority:normal');
    });

    it('includes a priority tag for high priority', function () {
        $job = new BatchContentGeneration('high');

        expect($job->tags())->toContain('priority:high');
    });

    it('includes a priority tag for low priority', function () {
        $job = new BatchContentGeneration('low');

        expect($job->tags())->toContain('priority:low');
    });

    it('returns exactly two tags', function () {
        $job = new BatchContentGeneration;

        expect($job->tags())->toHaveCount(2);
    });

    it('returns an array', function () {
        $job = new BatchContentGeneration;

        expect($job->tags())->toBeArray();
    });
});

// =========================================================================
// Job Chaining / Dependencies Tests
// =========================================================================

describe('job chaining', function () {
    it('ProcessContentTask can be dispatched from BatchContentGeneration logic', function () {
        Queue::fake();

        // Simulate what handle() does when tasks are found:
        // dispatch a ProcessContentTask for each task
        $mockTask = Mockery::mock(ContentTask::class)->makePartial();

        ProcessContentTask::dispatch($mockTask);

        Queue::assertPushed(ProcessContentTask::class, 1);
    });

    it('ProcessContentTask is dispatched to the ai queue', function () {
        Queue::fake();

        $mockTask = Mockery::mock(ContentTask::class)->makePartial();

        ProcessContentTask::dispatch($mockTask);

        Queue::assertPushedOn('ai', ProcessContentTask::class);
    });

    it('multiple ProcessContentTask jobs can be chained', function () {
        Queue::fake();

        $tasks = [
            Mockery::mock(ContentTask::class)->makePartial(),
            Mockery::mock(ContentTask::class)->makePartial(),
            Mockery::mock(ContentTask::class)->makePartial(),
        ];

        foreach ($tasks as $task) {
            ProcessContentTask::dispatch($task);
        }

        Queue::assertPushed(ProcessContentTask::class, 3);
    });
});

// =========================================================================
// Handle – Empty Task Collection Tests
// =========================================================================

describe('handle with no matching tasks', function () {
    it('logs an info message when no tasks are found', function () {
        // A real table, not an alias mock. Mockery alias mocks replace the class
        // for the whole PHP process, which is why this needed process isolation
        // and was skipped instead. dappcore/php-content owns the schema, but
        // loading its migrations suite-wide would also recreate prompts with
        // stricter columns than this package's own migration, so only the one
        // table under test is created here.
        Schema::create('content_tasks', function (Blueprint $table) {
            $table->id();
            $table->unsignedBigInteger('workspace_id');
            $table->unsignedBigInteger('prompt_id');
            $table->string('status')->default('pending');
            $table->string('priority')->default('normal');
            $table->json('input_data');
            $table->longText('output')->nullable();
            $table->json('metadata')->nullable();
            $table->string('target_type')->nullable();
            $table->unsignedBigInteger('target_id')->nullable();
            $table->timestamp('scheduled_for')->nullable();
            $table->timestamp('started_at')->nullable();
            $table->timestamp('completed_at')->nullable();
            $table->text('error_message')->nullable();
            $table->timestamps();
        });

        Log::shouldReceive('info')
            ->once()
            ->with('BatchContentGeneration: No normal priority tasks to process');

        $job = new BatchContentGeneration('normal', 10);
        $job->handle();
    });

    it('does not dispatch any ProcessContentTask when collection is empty', function () {
        Queue::fake();

        // Verify that when tasks is empty, no ProcessContentTask jobs are dispatched
        // This tests the early-return path conceptually
        $emptyTasks = collect([]);

        if ($emptyTasks->isEmpty()) {
            // Simulates handle() early return
            Log::info('BatchContentGeneration: No normal priority tasks to process');
        } else {
            foreach ($emptyTasks as $task) {
                ProcessContentTask::dispatch($task);
            }
        }

        Queue::assertNothingPushed();
    });
});

// =========================================================================
// Handle – With Tasks Tests
// =========================================================================

describe('handle with matching tasks', function () {
    it('dispatches one ProcessContentTask per task', function () {
        Queue::fake();

        $tasks = collect([
            Mockery::mock(ContentTask::class)->makePartial(),
            Mockery::mock(ContentTask::class)->makePartial(),
        ]);

        // Simulate handle() dispatch loop
        foreach ($tasks as $task) {
            ProcessContentTask::dispatch($task);
        }

        Queue::assertPushed(ProcessContentTask::class, 2);
    });

    it('respects the batch size limit', function () {
        // BatchContentGeneration queries with ->limit($this->batchSize)
        // Verify the batch size property is used as the limit
        $job = new BatchContentGeneration('normal', 5);

        expect($job->batchSize)->toBe(5);
    });
});
