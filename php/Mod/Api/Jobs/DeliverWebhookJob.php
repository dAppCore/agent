<?php

// SPDX-License-Identifier: EUPL-1.2

declare(strict_types=1);

namespace Core\Mod\Agentic\Mod\Api\Jobs;

use Core\Mod\Agentic\Mod\Api\Models\WebhookDelivery;
use Illuminate\Bus\Queueable;
use Illuminate\Contracts\Queue\ShouldQueue;
use Illuminate\Foundation\Bus\Dispatchable;
use Illuminate\Http\Client\ConnectionException;
use Illuminate\Queue\InteractsWithQueue;
use Illuminate\Queue\SerializesModels;
use Illuminate\Support\Facades\Http;

class DeliverWebhookJob implements ShouldQueue
{
    use Dispatchable;
    use InteractsWithQueue;
    use Queueable;
    use SerializesModels;

    public bool $deleteWhenMissingModels = true;

    public int $tries = 1;

    public function __construct(
        public WebhookDelivery $delivery
    ) {}

    public function handle(): void
    {
        $delivery = $this->delivery->fresh(['endpoint']);

        if (! $delivery instanceof WebhookDelivery) {
            return;
        }

        $endpoint = $delivery->endpoint;
        if ($endpoint === null || ! $endpoint->shouldReceive($delivery->event_type)) {
            $delivery->forceFill(['status' => WebhookDelivery::STATUS_CANCELLED])->save();

            return;
        }

        $delivery->forceFill(['status' => WebhookDelivery::STATUS_QUEUED])->save();
        $payload = $delivery->getDeliveryPayload();

        try {
            $response = Http::timeout(10)
                ->withHeaders($payload['headers'])
                ->withBody($payload['body'], 'application/json')
                ->post($endpoint->url);

            if ($response->successful()) {
                $delivery->markSuccess($response->status(), $response->body());

                return;
            }

            $this->handleFailure($delivery, $response->status(), $response->body());
        } catch (ConnectionException $exception) {
            $this->handleFailure($delivery, 0, 'Connection failed: '.$exception->getMessage());
        } catch (\Throwable $exception) {
            $this->handleFailure($delivery, 0, 'Unexpected error: '.$exception->getMessage());
        }
    }

    protected function handleFailure(WebhookDelivery $delivery, int $statusCode, ?string $responseBody): void
    {
        $delivery->markFailed($statusCode, $responseBody);
        $delivery->refresh();

        if (! $delivery->canRetry() || $delivery->next_retry_at === null) {
            return;
        }

        self::dispatch($delivery->fresh())->delay($delivery->next_retry_at);
    }
}
