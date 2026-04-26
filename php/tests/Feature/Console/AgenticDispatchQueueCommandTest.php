<?php

// SPDX-License-Identifier: EUPL-1.2

declare(strict_types=1);

use Core\Mod\Agentic\Console\Commands\AgenticDispatchQueueCommand;
use Core\Mod\Agentic\Jobs\DispatchMantisTicketJob;
use Core\Mod\Agentic\Models\AgentProfile;
use Illuminate\Console\Command;
use Illuminate\Contracts\Console\Kernel;
use Illuminate\Http\Client\Request;
use Illuminate\Support\Carbon;
use Illuminate\Support\Facades\Http;
use Illuminate\Support\Facades\Queue;

beforeEach(function (): void {
    if (! is_string(config('app.key')) || config('app.key') === '') {
        config(['app.key' => 'base64:'.base64_encode(random_bytes(32))]);
    }

    config([
        'agentic.mantis.base_url' => 'https://tasks.example.test',
        'agentic.mantis.token' => 'mantis-token-123',
    ]);

    Carbon::setTestNow('2026-04-26 12:00:00');

    $this->app->make(Kernel::class)->registerCommand(
        $this->app->make(AgenticDispatchQueueCommand::class),
    );
});

afterEach(function (): void {
    Carbon::setTestNow();
});

function dispatchQueueProfile(array $attributes = []): AgentProfile
{
    return AgentProfile::create(array_merge([
        'name' => $attributes['name'] ?? 'dispatch-profile',
        'gateway_url' => $attributes['gateway_url'] ?? 'https://gateway.example.test',
        'api_key_cipher' => $attributes['api_key_cipher'] ?? 'plain-secret',
        'cost_class' => 'C',
        'capability_tags' => ['dispatch'],
        'quota_headroom_pct' => 100,
        'enabled' => true,
        'last_dispatched_at' => null,
    ], $attributes));
}

test('AgenticDispatchQueueCommand_handle_Good_queues_up_to_the_default_limit_and_marks_tickets_in_flight', function (): void {
    Queue::fake();

    dispatchQueueProfile([
        'name' => 'core-profile',
        'cost_class' => 'A',
        'capability_tags' => ['core'],
    ]);

    dispatchQueueProfile([
        'name' => 'review-profile',
        'cost_class' => 'B',
        'capability_tags' => ['review'],
        'quota_headroom_pct' => 50,
    ]);

    dispatchQueueProfile([
        'name' => 'general-profile',
        'cost_class' => 'C',
        'capability_tags' => ['dispatch'],
    ]);

    Http::fake([
        'https://tasks.example.test/api/rest/issues?status=new' => Http::response([
            'issues' => [
                [
                    'id' => 900,
                    'handler' => [
                        'id' => 7,
                        'name' => 'assigned-user',
                    ],
                    'summary' => 'Already assigned ticket',
                    'tags' => ['core'],
                ],
                [
                    'id' => 901,
                    'notes' => [
                        [
                            'text' => 'Picked up by agentic dispatch queue (profile: stale-profile). Suppressing concurrent dispatch for 5min.',
                            'created_at' => now()->subMinutes(2)->toIso8601String(),
                        ],
                    ],
                    'summary' => 'Recently claimed ticket',
                    'tags' => ['review'],
                ],
                [
                    'id' => 902,
                    'severity' => 'critical',
                    'summary' => 'Core escalation',
                    'tags' => ['core'],
                ],
                [
                    'id' => 903,
                    'severity' => 'major',
                    'summary' => 'Review queue item',
                    'tags' => ['review'],
                ],
                [
                    'id' => 904,
                    'severity' => 'minor',
                    'summary' => 'General maintenance',
                    'tags' => [],
                ],
            ],
        ], 200),
        'https://tasks.example.test/api/rest/issues/*/notes' => Http::response([], 201),
    ]);

    $this->artisan('agentic:dispatch-queue')
        ->expectsOutputToContain('Queued 3 Mantis ticket(s); skipped 1 assigned, 1 in-flight, 0 without matching profile; 0 failed.')
        ->assertSuccessful();

    Queue::assertPushedOn('ai', DispatchMantisTicketJob::class);
    Queue::assertPushed(DispatchMantisTicketJob::class, 3);
    Queue::assertPushed(DispatchMantisTicketJob::class, fn (DispatchMantisTicketJob $job): bool => $job->ticketId === 902);
    Queue::assertPushed(DispatchMantisTicketJob::class, fn (DispatchMantisTicketJob $job): bool => $job->ticketId === 903);
    Queue::assertPushed(DispatchMantisTicketJob::class, fn (DispatchMantisTicketJob $job): bool => $job->ticketId === 904);
    Queue::assertNotPushed(DispatchMantisTicketJob::class, fn (DispatchMantisTicketJob $job): bool => in_array($job->ticketId, [900, 901], true));

    Http::assertSent(fn (Request $request): bool => $request->method() === 'GET'
        && $request->url() === 'https://tasks.example.test/api/rest/issues?status=new'
        && $request->hasHeader('Authorization', 'mantis-token-123'));

    Http::assertSent(fn (Request $request): bool => $request->method() === 'POST'
        && $request->url() === 'https://tasks.example.test/api/rest/issues/902/notes'
        && $request['text'] === 'Picked up by agentic dispatch queue (profile: core-profile). Suppressing concurrent dispatch for 5min.');

    Http::assertSent(fn (Request $request): bool => $request->method() === 'POST'
        && $request->url() === 'https://tasks.example.test/api/rest/issues/903/notes'
        && $request['text'] === 'Picked up by agentic dispatch queue (profile: review-profile). Suppressing concurrent dispatch for 5min.');

    Http::assertSent(fn (Request $request): bool => $request->method() === 'POST'
        && $request->url() === 'https://tasks.example.test/api/rest/issues/904/notes'
        && $request['text'] === 'Picked up by agentic dispatch queue (profile: general-profile). Suppressing concurrent dispatch for 5min.');

    Http::assertSentCount(4);
});

test('AgenticDispatchQueueCommand_handle_Bad_rejects_non_positive_limits', function (): void {
    Queue::fake();
    Http::fake();

    $this->artisan('agentic:dispatch-queue', ['--limit' => 0])
        ->expectsOutput('--limit must be greater than zero.')
        ->assertExitCode(Command::FAILURE);

    Queue::assertNothingPushed();
    Http::assertNothingSent();
});

test('AgenticDispatchQueueCommand_handle_Ugly_honours_a_custom_limit_after_skips', function (): void {
    Queue::fake();

    dispatchQueueProfile([
        'name' => 'core-profile',
        'cost_class' => 'A',
        'capability_tags' => ['core'],
    ]);

    dispatchQueueProfile([
        'name' => 'general-profile',
        'cost_class' => 'C',
        'capability_tags' => ['dispatch'],
    ]);

    Http::fake([
        'https://tasks.example.test/api/rest/issues?status=new' => Http::response([
            'issues' => [
                [
                    'id' => 910,
                    'handler' => [
                        'id' => 3,
                        'name' => 'assigned-user',
                    ],
                    'summary' => 'Assigned first',
                    'tags' => ['core'],
                ],
                [
                    'id' => 911,
                    'severity' => 'critical',
                    'summary' => 'First queueable ticket',
                    'tags' => ['core'],
                ],
                [
                    'id' => 912,
                    'severity' => 'minor',
                    'summary' => 'Would queue next if limit allowed',
                    'tags' => [],
                ],
            ],
        ], 200),
        'https://tasks.example.test/api/rest/issues/*/notes' => Http::response([], 201),
    ]);

    $this->artisan('agentic:dispatch-queue', ['--limit' => 1])
        ->expectsOutputToContain('Queued 1 Mantis ticket(s); skipped 1 assigned, 0 in-flight, 0 without matching profile; 0 failed.')
        ->assertSuccessful();

    Queue::assertPushed(DispatchMantisTicketJob::class, 1);
    Queue::assertPushed(DispatchMantisTicketJob::class, fn (DispatchMantisTicketJob $job): bool => $job->ticketId === 911);
    Queue::assertNotPushed(DispatchMantisTicketJob::class, fn (DispatchMantisTicketJob $job): bool => $job->ticketId === 912);

    Http::assertSentCount(2);
});
