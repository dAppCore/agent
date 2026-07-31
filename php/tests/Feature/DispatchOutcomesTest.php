<?php

/*
 * Core PHP Framework
 *
 * Licensed under the European Union Public Licence (EUPL) v1.2.
 * See LICENSE file for details.
 */

declare(strict_types=1);

use Core\Mod\Agentic\Models\AgentRegistration;
use Core\Mod\Agentic\Models\DispatchJob;
use Core\Mod\Agentic\Services\DispatchService;
use Core\Tenant\Models\Workspace;
use Illuminate\Support\Facades\Schema;

function dispatchService(): DispatchService
{
    return app(DispatchService::class);
}

/**
 * A job already claimed by an agent, which is the state every outcome acts on.
 *
 * @return array{0: Workspace, 1: DispatchJob, 2: string}
 */
function claimedJob(?Workspace $workspace = null, string $agentId = 'agent-1'): array
{
    $workspace ??= Workspace::factory()->create();

    $job = DispatchJob::query()->create([
        'workspace_id' => $workspace->id,
        'repo' => 'dappcore/go-inference',
        'task' => 'Fix the thing',
        'status' => DispatchJob::STATUS_RUNNING,
        'assigned_agent' => $agentId,
        'assigned_at' => now(),
    ]);

    // The model mints its own uuid (HasUuids), so the registration has to point
    // at the id the job actually got rather than one chosen beforehand.
    AgentRegistration::query()->create([
        'workspace_id' => $workspace->id,
        'agent_id' => $agentId,
        'current_task_id' => $job->id,
    ]);

    return [$workspace, $job, $agentId];
}

function registrationTask(Workspace $workspace, string $agentId): ?string
{
    return AgentRegistration::query()
        ->where('workspace_id', $workspace->id)
        ->where('agent_id', $agentId)
        ->value('current_task_id');
}

// ─── The table exists at all ─────────────────────────────────────────────

it('has a table for the model it ships', function () {
    // The model shipped without a migration, so every consumer that touched
    // DispatchJob hit a missing table.
    expect(Schema::hasTable('dispatch_jobs'))->toBeTrue();
});

it('stores every column the model declares fillable', function () {
    [$workspace] = claimedJob();

    $job = DispatchJob::query()->create([
        'workspace_id' => $workspace->id,
        'created_by' => 'cladius',
        'repo' => 'dappcore/go-html',
        'org' => 'dappcore',
        'task' => 'everything',
        'agent_type' => 'codex',
        'template' => 'bug-fix',
        'branch' => 'dev',
        'priority' => 1,
        'labels' => ['fix'],
        'status' => DispatchJob::STATUS_PENDING,
        'findings' => ['a'],
        'changes' => ['b'],
        'report' => ['c'],
        'metadata' => ['d' => 1],
    ]);

    $fresh = $job->fresh();

    expect($fresh->created_by)->toBe('cladius')
        ->and($fresh->findings)->toBe(['a'])
        ->and($fresh->changes)->toBe(['b'])
        ->and($fresh->report)->toBe(['c'])
        ->and($fresh->priority)->toBe(1);
});

// ─── Failing ─────────────────────────────────────────────────────────────

it('records a failure with the reason that caused it', function () {
    [$workspace, $job, $agentId] = claimedJob();

    $failed = dispatchService()->fail($workspace->id, $agentId, $job->id, 'compiler ran out of memory');

    expect($failed->status)->toBe(DispatchJob::STATUS_FAILED)
        ->and($failed->metadata['failure_reason'])->toBe('compiler ran out of memory')
        ->and($failed->metadata['failed_by'])->toBe($agentId)
        ->and($failed->completed_at)->not->toBeNull();
});

it('frees the agent when its job fails', function () {
    [$workspace, $job, $agentId] = claimedJob();

    expect(registrationTask($workspace, $agentId))->toBe($job->id);

    dispatchService()->fail($workspace->id, $agentId, $job->id, 'no');

    // Otherwise a failure leaves the agent looking permanently busy and it
    // never gets given anything else.
    expect(registrationTask($workspace, $agentId))->toBeNull();
});

it('will not let an agent fail a job it does not hold', function () {
    [$workspace, $job] = claimedJob();

    expect(dispatchService()->fail($workspace->id, 'someone-else', $job->id, 'not mine'))->toBeNull()
        ->and($job->fresh()->status)->toBe(DispatchJob::STATUS_RUNNING);
});

it('will not let one workspace fail another workspace\'s job', function () {
    [, $job, $agentId] = claimedJob();
    $other = Workspace::factory()->create();

    expect(dispatchService()->fail($other->id, $agentId, $job->id, 'nice try'))->toBeNull()
        ->and($job->fresh()->status)->toBe(DispatchJob::STATUS_RUNNING);
});

// ─── Cancelling ──────────────────────────────────────────────────────────

it('cancels a job without needing to be the agent holding it', function () {
    [$workspace, $job, $agentId] = claimedJob();

    // Cancelling is the workspace's decision, not the agent's.
    $cancelled = dispatchService()->cancel($workspace->id, $job->id, 'no longer needed');

    expect($cancelled->status)->toBe(DispatchJob::STATUS_CANCELLED)
        ->and($cancelled->metadata['cancel_reason'])->toBe('no longer needed')
        ->and(registrationTask($workspace, $agentId))->toBeNull();
});

it('cancels an unassigned job too', function () {
    $workspace = Workspace::factory()->create();
    $job = DispatchJob::query()->create([
        'workspace_id' => $workspace->id,
        'repo' => 'dappcore/cli',
        'task' => 'never picked up',
        'status' => DispatchJob::STATUS_PENDING,
    ]);

    expect(dispatchService()->cancel($workspace->id, $job->id)->status)
        ->toBe(DispatchJob::STATUS_CANCELLED);
});

it('leaves finished work alone', function () {
    [$workspace, $job, $agentId] = claimedJob();
    dispatchService()->complete($workspace->id, $agentId, $job->id);

    // Cancelling after the fact would rewrite the record of what happened.
    $result = dispatchService()->cancel($workspace->id, $job->id, 'too late');

    expect($result->status)->toBe(DispatchJob::STATUS_COMPLETED)
        ->and($result->metadata['cancel_reason'] ?? null)->toBeNull();
});

it('will not cancel another workspace\'s job', function () {
    [, $job] = claimedJob();
    $other = Workspace::factory()->create();

    expect(dispatchService()->cancel($other->id, $job->id))->toBeNull()
        ->and($job->fresh()->status)->toBe(DispatchJob::STATUS_RUNNING);
});

it('distinguishes cancelled from failed', function () {
    [$w1, $j1, $a1] = claimedJob();
    [$w2, $j2] = claimedJob(agentId: 'agent-2');

    dispatchService()->fail($w1->id, $a1, $j1->id, 'broke');
    dispatchService()->cancel($w2->id, $j2->id);

    // A failure is worth investigating; a cancellation is not. Collapsing them
    // would lose that.
    expect($j1->fresh()->status)->toBe(DispatchJob::STATUS_FAILED)
        ->and($j2->fresh()->status)->toBe(DispatchJob::STATUS_CANCELLED);
});
