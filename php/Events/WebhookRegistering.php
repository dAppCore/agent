<?php

// SPDX-License-Identifier: EUPL-1.2

declare(strict_types=1);

namespace Core\Events;

/**
 * Fired when webhook ingress types are being registered.
 *
 * Modules and external applications listen to this event to publish the
 * webhook types they want Core to expose during application boot.
 *
 * ## When This Event Fires
 *
 * Fired once the application has booted and webhook ingress definitions are
 * being collected for the active runtime.
 *
 * ## Usage Example
 *
 * ```php
 * public static array $listens = [
 *     WebhookRegistering::class => 'onWebhookRegistering',
 * ];
 *
 * public function onWebhookRegistering(WebhookRegistering $event): void
 * {
 *     $event->register('myapp.event.x', [
 *         'payload' => [
 *             'id' => 'string',
 *         ],
 *     ]);
 * }
 * ```
 */
class WebhookRegistering extends LifecycleEvent
{
    /** @var array<string, array<string, mixed>> Collected webhook type definitions keyed by type */
    protected array $types = [];

    /**
     * Register a webhook type definition.
     *
     * Later registrations with the same type replace earlier definitions so an
     * application can override defaults during boot.
     *
     * @param  string  $type  Fully qualified webhook type name
     * @param  array<string, mixed>  $definition  Payload shape or metadata for the type
     */
    public function register(string $type, array $definition = []): void
    {
        $this->types[$type] = $definition;
    }

    /**
     * Get all registered webhook type definitions.
     *
     * @return array<string, array<string, mixed>>
     *
     * @internal Used by the webhook ingress bootstrap
     */
    public function types(): array
    {
        return $this->types;
    }
}
