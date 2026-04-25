<?php

// SPDX-License-Identifier: EUPL-1.2

declare(strict_types=1);

namespace Core\Mod\Agentic\Mod\Api\Services;

use Core\Mod\Agentic\Mod\Api\Jobs\DeliverWebhookJob;
use Core\Mod\Agentic\Mod\Api\Models\WebhookDelivery;
use Core\Mod\Agentic\Mod\Api\Models\WebhookEndpoint;
use Illuminate\Support\Facades\DB;

class WebhookService
{
    /**
     * Dispatch an event to every active endpoint subscribed to the event type.
     *
     * Example:
     * `dispatch(12, 'mcp.tool.executed', ['tool' => 'brain_recall'])`
     *
     * @return array<int, WebhookDelivery>
     */
    public function dispatch(int $workspaceId, string $eventType, array $data): array
    {
        $endpoints = WebhookEndpoint::query()
            ->forWorkspace($workspaceId)
            ->active()
            ->forEvent($eventType)
            ->get();

        if ($endpoints->isEmpty()) {
            return [];
        }

        $deliveries = [];

        DB::transaction(function () use ($data, $endpoints, $eventType, $workspaceId, &$deliveries): void {
            foreach ($endpoints as $endpoint) {
                $delivery = WebhookDelivery::createForEvent($endpoint, $eventType, $data, $workspaceId);
                $deliveries[] = $delivery;

                DeliverWebhookJob::dispatch($delivery)->afterCommit();
            }
        });

        return $deliveries;
    }
}
