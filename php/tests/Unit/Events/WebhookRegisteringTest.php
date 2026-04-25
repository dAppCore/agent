<?php

// SPDX-License-Identifier: EUPL-1.2

declare(strict_types=1);

use Core\CoreServiceProvider;
use Core\Events\WebhookRegistering;
use Illuminate\Support\Facades\Event;

if (! class_exists(WebhookRegistering::class)) {
    require_once dirname(__DIR__, 3).'/Events/WebhookRegistering.php';
}

if (! class_exists(CoreServiceProvider::class)) {
    require_once dirname(__DIR__, 3).'/CoreServiceProvider.php';
}

test('WebhookRegistering_register_Good_collects_webhook_type_definitions', function (): void {
    $event = new WebhookRegistering;

    $event->register('myapp.event.x', [
        'payload' => [
            'id' => 'string',
        ],
    ]);

    expect($event->types())->toBe([
        'myapp.event.x' => [
            'payload' => [
                'id' => 'string',
            ],
        ],
    ]);
});

test('CoreServiceProvider_boot_Good_dispatches_webhook_registering', function (): void {
    $received = false;

    Event::listen(WebhookRegistering::class, function (WebhookRegistering $event) use (&$received): void {
        $received = true;

        $event->register('ofm.bot.message.received', [
            'payload' => [
                'message_id' => 'string',
            ],
        ]);
    });

    $this->app->register(CoreServiceProvider::class);

    expect($received)->toBeTrue();
});

test('CoreServiceProvider_webhookTypes_Good_returns_registered_types_after_boot', function (): void {
    Event::listen(WebhookRegistering::class, function (WebhookRegistering $event): void {
        $event->register('ofm.bot.message.received', [
            'payload' => [
                'message_id' => 'string',
            ],
        ]);

        $event->register('ofm.bot.message.deleted', [
            'payload' => [
                'message_id' => 'string',
            ],
        ]);
    });

    $this->app->register(CoreServiceProvider::class);

    expect(CoreServiceProvider::webhookTypes())->toBe([
        'ofm.bot.message.received' => [
            'payload' => [
                'message_id' => 'string',
            ],
        ],
        'ofm.bot.message.deleted' => [
            'payload' => [
                'message_id' => 'string',
            ],
        ],
    ]);
});
