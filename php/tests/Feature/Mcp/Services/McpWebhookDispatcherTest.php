<?php

// SPDX-License-Identifier: EUPL-1.2

declare(strict_types=1);

require_once dirname(__DIR__).'/Support/bootstrap.php';

mcpRequire('Mcp/Services/McpWebhookDispatcher.php');

use Core\Mcp\Services\McpWebhookDispatcher;
use Illuminate\Support\Facades\Http;

final class McpWebhookDispatcherEndpointBuilderStub
{
    public function __construct(private array $records)
    {
    }

    public function getModel(): object
    {
        return new McpWebhookDispatcherEndpointStub([]);
    }

    public function forWorkspace(int $workspaceId): self
    {
        $this->records = array_values(array_filter(
            $this->records,
            static fn (McpWebhookDispatcherEndpointStub $endpoint): bool => $endpoint->workspace_id === $workspaceId,
        ));

        return $this;
    }

    public function active(): self
    {
        $this->records = array_values(array_filter(
            $this->records,
            static fn (McpWebhookDispatcherEndpointStub $endpoint): bool => $endpoint->active,
        ));

        return $this;
    }

    public function forEvent(string $eventType): self
    {
        $this->records = array_values(array_filter(
            $this->records,
            static fn (McpWebhookDispatcherEndpointStub $endpoint): bool => in_array($eventType, $endpoint->events, true),
        ));

        return $this;
    }

    public function get(): array
    {
        return $this->records;
    }
}

final class McpWebhookDispatcherEndpointStub
{
    public static array $records = [];

    public function __construct(
        public array $events,
        public int $workspace_id = 1,
        public bool $active = true,
        public string $url = 'https://hooks.example.test/mcp',
        public int $id = 1,
        public int $successes = 0,
        public int $failures = 0,
    ) {
    }

    public static function query(): McpWebhookDispatcherEndpointBuilderStub
    {
        return new McpWebhookDispatcherEndpointBuilderStub(static::$records);
    }

    public function generateSignature(string $payload): string
    {
        return 'sig:'.sha1($payload);
    }

    public function recordSuccess(): void
    {
        $this->successes++;
    }

    public function recordFailure(): void
    {
        $this->failures++;
    }
}

final class McpWebhookDispatcherDeliveryStub
{
    public static array $records = [];

    public static function create(array $attributes): void
    {
        static::$records[] = $attributes;
    }
}

beforeEach(function (): void {
    Http::preventStrayRequests();
    McpWebhookDispatcherEndpointStub::$records = [];
    McpWebhookDispatcherDeliveryStub::$records = [];
});

test('McpWebhookDispatcher_dispatchToolExecuted_Good_delivers_to_matching_endpoints_and_records_success', function (): void {
    McpWebhookDispatcherEndpointStub::$records = [
        new McpWebhookDispatcherEndpointStub(['mcp.tool.executed']),
    ];

    Http::fake([
        'https://hooks.example.test/mcp' => Http::response('ok', 200),
    ]);

    $dispatcher = new class extends McpWebhookDispatcher
    {
        protected function endpointModelClass(): ?string
        {
            return McpWebhookDispatcherEndpointStub::class;
        }

        protected function deliveryModelClass(): ?string
        {
            return McpWebhookDispatcherDeliveryStub::class;
        }
    };

    $dispatcher->dispatchToolExecuted(1, 'host-hub', 'send_email', ['to' => 'a@example.com'], true, 42);

    expect(McpWebhookDispatcherDeliveryStub::$records)->toHaveCount(1)
        ->and(McpWebhookDispatcherDeliveryStub::$records[0]['status'])->toBe('success')
        ->and(McpWebhookDispatcherEndpointStub::$records[0]->successes)->toBe(1);
});

test('McpWebhookDispatcher_dispatchToolExecuted_Bad_noops_when_no_endpoints_are_subscribed', function (): void {
    $dispatcher = new class extends McpWebhookDispatcher
    {
        protected function endpointModelClass(): ?string
        {
            return McpWebhookDispatcherEndpointStub::class;
        }

        protected function deliveryModelClass(): ?string
        {
            return McpWebhookDispatcherDeliveryStub::class;
        }
    };

    $dispatcher->dispatchToolExecuted(1, 'host-hub', 'send_email', [], true, 10);

    expect(McpWebhookDispatcherDeliveryStub::$records)->toHaveCount(0);
});

test('McpWebhookDispatcher_dispatchToolExecuted_Ugly_records_failed_deliveries_and_failure_counts', function (): void {
    McpWebhookDispatcherEndpointStub::$records = [
        new McpWebhookDispatcherEndpointStub(['mcp.tool.executed']),
    ];

    Http::fake([
        'https://hooks.example.test/mcp' => Http::response('broken', 500),
    ]);

    $dispatcher = new class extends McpWebhookDispatcher
    {
        protected function endpointModelClass(): ?string
        {
            return McpWebhookDispatcherEndpointStub::class;
        }

        protected function deliveryModelClass(): ?string
        {
            return McpWebhookDispatcherDeliveryStub::class;
        }
    };

    $dispatcher->dispatchToolExecuted(1, 'host-hub', 'send_email', [], false, 10, 'failed');

    expect(McpWebhookDispatcherDeliveryStub::$records[0]['status'])->toBe('failed')
        ->and(McpWebhookDispatcherEndpointStub::$records[0]->failures)->toBe(1);
});
