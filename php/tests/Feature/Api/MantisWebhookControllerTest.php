<?php

// SPDX-License-Identifier: EUPL-1.2

declare(strict_types=1);

use Core\Mod\Agentic\Jobs\DispatchMantisTicketJob;
use Core\Mod\Agentic\Models\AgentProfile;
use Illuminate\Support\Facades\Queue;
use Illuminate\Support\Facades\Route;

beforeEach(function (): void {
    if (! is_string(config('app.key')) || config('app.key') === '') {
        config(['app.key' => 'base64:'.base64_encode(random_bytes(32))]);
    }

    config(['agentic.mantis.webhook_secret' => 'mantis-secret']);

    Route::prefix('api')->group(function (): void {
        require __DIR__.'/../../../Routes/api.php';
    });
});

function mantisWebhookProfile(array $overrides = []): AgentProfile
{
    static $sequence = 0;

    $sequence++;

    return AgentProfile::query()->create(array_merge([
        'name' => "mantis-webhook-profile-{$sequence}",
        'gateway_url' => "https://gateway-{$sequence}.example.test",
        'api_key_cipher' => "secret-{$sequence}",
        'cost_class' => 'C',
        'capability_tags' => ['dispatch'],
        'quota_headroom_pct' => 100,
        'enabled' => true,
        'last_dispatched_at' => null,
    ], $overrides));
}

test('MantisWebhookController_receive_Good_returns_200_and_dispatches_issue_opened', function (): void {
    Queue::fake();

    mantisWebhookProfile();

    $response = $this
        ->withHeader('X-Mantis-Webhook-Secret', 'mantis-secret')
        ->postJson('/api/agentic/mantis-webhook', [
            'event' => 'issue.opened',
            'issue' => [
                'id' => 123,
                'summary' => 'Investigate the regression',
                'severity' => 'minor',
                'priority' => 'normal',
            ],
        ]);

    $response
        ->assertOk()
        ->assertJsonPath('status', 'accepted')
        ->assertJsonPath('dispatched', true);

    Queue::assertPushedOn('ai', DispatchMantisTicketJob::class);
    Queue::assertPushed(DispatchMantisTicketJob::class, function (DispatchMantisTicketJob $job): bool {
        return $job->ticketId === 123;
    });
});

test('MantisWebhookController_receive_Bad_returns_401_for_a_wrong_secret', function (): void {
    Queue::fake();

    $response = $this
        ->withHeader('X-Mantis-Webhook-Secret', 'wrong-secret')
        ->postJson('/api/agentic/mantis-webhook', [
            'event' => 'issue.opened',
            'issue' => [
                'id' => 123,
            ],
        ]);

    $response
        ->assertUnauthorized()
        ->assertJsonPath('message', 'Unauthorised');

    Queue::assertNothingPushed();
});

test('MantisWebhookController_receive_Good_returns_204_for_an_unhandled_event', function (): void {
    Queue::fake();

    mantisWebhookProfile();

    $response = $this
        ->withHeader('X-Mantis-Webhook-Secret', 'mantis-secret')
        ->postJson('/api/agentic/mantis-webhook', [
            'event' => 'issue.closed',
            'issue' => [
                'id' => 123,
                'summary' => 'Closed by maintainer',
            ],
        ]);

    $response->assertNoContent();

    Queue::assertNothingPushed();
});

test('MantisWebhookController_receive_Ugly_returns_422_for_a_malformed_body', function (): void {
    Queue::fake();

    $response = $this
        ->withHeader('X-Mantis-Webhook-Secret', 'mantis-secret')
        ->postJson('/api/agentic/mantis-webhook', [
            'event' => 'issue.opened',
            'issue' => [
                'summary' => 'Missing identifier',
            ],
        ]);

    $response
        ->assertUnprocessable()
        ->assertJsonValidationErrors(['issue.id']);

    Queue::assertNothingPushed();
});
