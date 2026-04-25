<?php

// SPDX-License-Identifier: EUPL-1.2

declare(strict_types=1);

use Core\Mod\Agentic\Jobs\CaptureDispatchResultJob;
use Core\Mod\Agentic\Jobs\DispatchMantisTicketJob;
use Core\Mod\Agentic\Models\AgentDispatch;
use Core\Mod\Agentic\Models\AgentProfile;
use Core\Mod\Agentic\Services\HermesClient;
use Core\Mod\Agentic\Services\MantisClient;
use Core\Mod\Agentic\Services\ProfileSelector;
use Illuminate\Bus\Queueable;
use Illuminate\Contracts\Queue\Job as QueueJobContract;
use Illuminate\Contracts\Queue\ShouldQueue;
use Illuminate\Foundation\Bus\Dispatchable;
use Illuminate\Queue\InteractsWithQueue;
use Illuminate\Queue\SerializesModels;
use Illuminate\Support\Facades\Http;
use Illuminate\Support\Facades\Log;
use Illuminate\Support\Facades\Queue;

if (! class_exists(CaptureDispatchResultJob::class)) {
    class DispatchMantisTicketJobTestCaptureDispatchResultJob implements ShouldQueue
    {
        use Dispatchable, InteractsWithQueue, Queueable, SerializesModels;

        public function __construct(
            public int $ticketId,
            public string $responseId,
        ) {}
    }

    class_alias(
        DispatchMantisTicketJobTestCaptureDispatchResultJob::class,
        CaptureDispatchResultJob::class,
    );
}

if (! class_exists(MantisClient::class)) {
    class DispatchMantisTicketJobTestMantisClient {}

    class_alias(
        DispatchMantisTicketJobTestMantisClient::class,
        MantisClient::class,
    );
}

if (! class_exists(ProfileSelector::class)) {
    class DispatchMantisTicketJobTestProfileSelector
    {
        public function pickFor(mixed $ticket): ?AgentProfile
        {
            return null;
        }
    }

    class_alias(
        DispatchMantisTicketJobTestProfileSelector::class,
        ProfileSelector::class,
    );
}

it('dispatches with the ticket identifier payload', function () {
    Queue::fake();

    DispatchMantisTicketJob::dispatch(827);

    Queue::assertPushedOn('ai', DispatchMantisTicketJob::class);
    Queue::assertPushed(DispatchMantisTicketJob::class, function ($queuedJob) {
        return $queuedJob->ticketId === 827;
    });
});

it('posts the prompt to Hermes, records the dispatch, and queues result capture', function () {
    Queue::fake();

    Http::fake([
        'https://hermes.example.test/v1/responses' => Http::response([
            'id' => 'resp_123',
            'run_id' => 'run_456',
            'status' => 'queued',
        ], 200),
    ]);

    $profile = AgentProfile::query()->create([
        'name' => 'Hermes Default',
        'gateway_url' => 'https://hermes.example.test',
        'api_key_cipher' => 'hermes-test-key',
        'cost_class' => 'm',
        'capability_tags' => ['mantis'],
        'quota_headroom_pct' => 100,
        'enabled' => true,
    ]);

    $ticket = [
        'body' => 'Investigate the failing regression test in the dispatch pipeline.',
    ];

    $mantisClient = Mockery::mock(MantisClient::class);
    $mantisClient->shouldReceive('get')
        ->once()
        ->with(827)
        ->andReturn($ticket);

    $profileSelector = Mockery::mock(ProfileSelector::class);
    $profileSelector->shouldReceive('pickFor')
        ->once()
        ->with($ticket)
        ->andReturn($profile);

    $this->app->instance(MantisClient::class, $mantisClient);
    $this->app->instance(ProfileSelector::class, $profileSelector);

    $job = new DispatchMantisTicketJob(827);

    $this->app->call([$job, 'handle'], [
        'hermesClient' => new HermesClient,
    ]);

    Http::assertSent(function ($request) {
        return $request->method() === 'POST'
            && $request->url() === 'https://hermes.example.test/v1/responses'
            && $request->hasHeader('Authorization', 'Bearer hermes-test-key')
            && $request['model'] === 'default'
            && $request['input'] === 'Investigate the failing regression test in the dispatch pipeline.'
            && $request['metadata']['conversation'] === 'mantis-827'
            && $request['metadata']['ticket_id'] === 827;
    });

    $dispatch = AgentDispatch::query()->sole();

    expect($dispatch->ticket_id)->toBe(827)
        ->and($dispatch->profile_id)->toBe($profile->id)
        ->and($dispatch->response_id)->toBe('resp_123')
        ->and($dispatch->run_id)->toBe('run_456')
        ->and($dispatch->status)->toBe('queued');

    Queue::assertPushed(CaptureDispatchResultJob::class, 1);
});

it('logs and releases the job when no profile is available', function () {
    Http::fake();

    $ticket = [
        'body' => 'Retry this ticket once a profile becomes available.',
    ];

    $mantisClient = Mockery::mock(MantisClient::class);
    $mantisClient->shouldReceive('get')
        ->once()
        ->with(912)
        ->andReturn($ticket);

    $profileSelector = Mockery::mock(ProfileSelector::class);
    $profileSelector->shouldReceive('pickFor')
        ->once()
        ->with($ticket)
        ->andReturn(null);

    Log::shouldReceive('warning')
        ->once()
        ->with('DispatchMantisTicketJob: no agent profile available for Mantis ticket', [
            'ticket_id' => 912,
        ]);

    $queuedJob = Mockery::mock(QueueJobContract::class);
    $queuedJob->shouldReceive('release')
        ->once()
        ->with(60);

    $job = new DispatchMantisTicketJob(912);
    $job->setJob($queuedJob);
    $job->handle($mantisClient, new HermesClient, $profileSelector);

    expect(AgentDispatch::query()->count())->toBe(0);

    Http::assertNothingSent();
});
