<?php

declare(strict_types=1);

use Carbon\Carbon;
use Core\Mod\Agentic\Mod\Api\Boot as ApiBoot;
use Core\Mod\Agentic\Mod\Api\Jobs\DeliverWebhookJob;
use Core\Mod\Agentic\Mod\Api\Models\WebhookDelivery;
use Core\Mod\Agentic\Mod\Api\Models\WebhookEndpoint;
use Core\Mod\Agentic\Mod\Api\Services\WebhookService;
use Core\Mod\Agentic\Mod\Api\Services\WebhookSignature;
use Illuminate\Support\Facades\Queue;

beforeEach(function (): void {
    $this->app->register(ApiBoot::class);
    $this->workspace = createWorkspace();
});

afterEach(function (): void {
    Carbon::setTestNow();
});

describe('Webhook foundation', function () {
    it('queues deliveries after the transaction commits', function (): void {
        Queue::fake();

        WebhookEndpoint::createForWorkspace(
            $this->workspace->id,
            'https://example.com/webhooks/core',
            ['mcp.tool.executed'],
        );

        $deliveries = app(WebhookService::class)->dispatch(
            $this->workspace->id,
            'mcp.tool.executed',
            ['tool' => 'brain_recall'],
        );

        expect($deliveries)->toHaveCount(1)
            ->and(WebhookDelivery::count())->toBe(1);

        Queue::assertPushed(DeliverWebhookJob::class, function (DeliverWebhookJob $job): bool {
            return $job->delivery->event_type === 'mcp.tool.executed';
        });
    });

    it('accepts both current and previous secrets during the rotation window', function (): void {
        $endpoint = WebhookEndpoint::createForWorkspace(
            $this->workspace->id,
            'https://example.com/webhooks/core',
            ['mcp.tool.executed'],
        );

        $payload = '{"tool":"brain_recall"}';
        $timestamp = time();
        $signer = app(WebhookSignature::class);
        $oldSecret = $endpoint->getRawOriginal('secret');
        $oldSignature = $signer->sign($payload, (string) $oldSecret, $timestamp);

        $newSecret = $endpoint->rotateSecret(300);
        $endpoint->refresh();

        expect($endpoint->verifySignature($payload, $oldSignature, $timestamp))->toBeTrue()
            ->and($endpoint->verifySignature($payload, $signer->sign($payload, $newSecret, $timestamp), $timestamp))->toBeTrue();

        $endpoint->update(['previous_secret_expires_at' => now()->subSecond()]);

        expect($endpoint->fresh()->verifySignature($payload, $oldSignature, $timestamp))->toBeFalse();
    });

    it('auto-disables endpoints after ten consecutive failures', function (): void {
        $endpoint = WebhookEndpoint::createForWorkspace(
            $this->workspace->id,
            'https://example.com/webhooks/core',
            ['mcp.tool.executed'],
        );

        foreach (range(1, 10) as $attempt) {
            $endpoint->recordFailure();
        }

        expect($endpoint->fresh()->failure_count)->toBe(10)
            ->and($endpoint->fresh()->is_active)->toBeFalse()
            ->and($endpoint->fresh()->disabled_at)->not->toBeNull();
    });

    it('schedules the first retry five minutes after a failed delivery', function (): void {
        Carbon::setTestNow(Carbon::parse('2026-04-25 12:00:00'));

        $endpoint = WebhookEndpoint::createForWorkspace(
            $this->workspace->id,
            'https://example.com/webhooks/core',
            ['mcp.tool.executed'],
        );

        $delivery = WebhookDelivery::createForEvent(
            $endpoint,
            'mcp.tool.executed',
            ['tool' => 'brain_recall'],
            $this->workspace->id,
        );

        $delivery->markFailed(500, 'Upstream timeout');

        expect($delivery->fresh()->attempt)->toBe(2)
            ->and($delivery->fresh()->status)->toBe(WebhookDelivery::STATUS_RETRYING)
            ->and($delivery->fresh()->next_retry_at?->equalTo(now()->addMinutes(5)))->toBeTrue();
    });
});
