<?php

// SPDX-License-Identifier: EUPL-1.2

declare(strict_types=1);

namespace Core;

use Core\Events\WebhookRegistering;
use Illuminate\Support\ServiceProvider;

/**
 * Core service provider shim for boot-time webhook type registration.
 *
 * Stores the collected webhook type registry in the container after the
 * application has fully booted so downstream services can resolve it.
 */
class CoreServiceProvider extends ServiceProvider
{
    public const WEBHOOK_TYPES = 'core.webhook.types';

    public function register(): void
    {
        $this->app->singleton(self::WEBHOOK_TYPES, static fn (): array => []);
    }

    public function boot(): void
    {
        $this->app->booted(function (): void {
            $this->app->instance(self::WEBHOOK_TYPES, static::fireWebhookRegistering());
        });
    }

    /**
     * Fire WebhookRegistering and return the collected type registry.
     *
     * @return array<string, array<string, mixed>>
     */
    public static function fireWebhookRegistering(): array
    {
        $event = new WebhookRegistering;
        event($event);

        return $event->types();
    }

    /**
     * Get the webhook type registry captured during boot.
     *
     * @return array<string, array<string, mixed>>
     */
    public static function webhookTypes(): array
    {
        return app()->bound(self::WEBHOOK_TYPES)
            ? app(self::WEBHOOK_TYPES)
            : [];
    }
}
