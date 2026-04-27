<?php

// SPDX-License-Identifier: EUPL-1.2

declare(strict_types=1);

namespace Core\Mod\Agentic\Mod\Api\Models;

use Core\Mod\Agentic\Mod\Api\Services\WebhookSignature;
use Core\Tenant\Models\Workspace;
use Illuminate\Database\Eloquent\Model;
use Illuminate\Database\Eloquent\Relations\BelongsTo;
use Illuminate\Database\Eloquent\Relations\HasMany;
use Illuminate\Database\Eloquent\SoftDeletes;

class WebhookEndpoint extends Model
{
    use SoftDeletes;

    protected $fillable = [
        'workspace_id',
        'url',
        'secret',
        'previous_secret',
        'previous_secret_expires_at',
        'events',
        'is_active',
        'description',
        'last_triggered_at',
        'failure_count',
        'disabled_at',
    ];

    protected $casts = [
        'events' => 'array',
        'is_active' => 'boolean',
        'previous_secret_expires_at' => 'datetime',
        'last_triggered_at' => 'datetime',
        'disabled_at' => 'datetime',
    ];

    protected $hidden = [
        'secret',
        'previous_secret',
    ];

    public static function createForWorkspace(
        int $workspaceId,
        string $url,
        array $events,
        ?string $description = null
    ): static {
        $signature = app(WebhookSignature::class);

        return static::query()->create([
            'workspace_id' => $workspaceId,
            'url' => $url,
            'secret' => $signature->generateSecret(),
            'events' => array_values($events),
            'is_active' => true,
            'description' => $description,
            'failure_count' => 0,
        ]);
    }

    public function shouldReceive(string $eventType): bool
    {
        if (! $this->is_active || $this->disabled_at !== null) {
            return false;
        }

        return in_array($eventType, $this->events ?? [], true)
            || in_array('*', $this->events ?? [], true);
    }

    public function generateSignature(string $payload, int $timestamp): string
    {
        return app(WebhookSignature::class)->sign($payload, $this->secret, $timestamp);
    }

    public function verifySignature(
        string $payload,
        string $signature,
        int $timestamp,
        int $tolerance = WebhookSignature::DEFAULT_TOLERANCE
    ): bool {
        $signer = app(WebhookSignature::class);

        if ($signer->verify($payload, $signature, $this->secret, $timestamp, $tolerance)) {
            return true;
        }

        if (! $this->hasPreviousSecret()) {
            return false;
        }

        return $signer->verify($payload, $signature, (string) $this->previous_secret, $timestamp, $tolerance);
    }

    public function rotateSecret(int $gracePeriodSeconds = 86400): string
    {
        $signer = app(WebhookSignature::class);
        $newSecret = $signer->generateSecret();

        $this->forceFill([
            'previous_secret' => $this->secret,
            'previous_secret_expires_at' => now()->addSeconds($gracePeriodSeconds),
            'secret' => $newSecret,
        ])->save();

        return $newSecret;
    }

    public function recordSuccess(): void
    {
        $this->forceFill([
            'last_triggered_at' => now(),
            'failure_count' => 0,
        ])->save();
    }

    public function recordFailure(): void
    {
        $failureCount = $this->failure_count + 1;

        $updates = [
            'failure_count' => $failureCount,
            'last_triggered_at' => now(),
        ];

        if ($failureCount >= 10) {
            $updates['is_active'] = false;
            $updates['disabled_at'] = now();
        }

        $this->forceFill($updates)->save();
    }

    public function enable(): void
    {
        $this->forceFill([
            'is_active' => true,
            'disabled_at' => null,
            'failure_count' => 0,
        ])->save();
    }

    public function hasPreviousSecret(): bool
    {
        return $this->previous_secret !== null
            && $this->previous_secret_expires_at !== null
            && $this->previous_secret_expires_at->isFuture();
    }

    public function workspace(): BelongsTo
    {
        return $this->belongsTo(Workspace::class, 'workspace_id');
    }

    public function deliveries(): HasMany
    {
        return $this->hasMany(WebhookDelivery::class, 'webhook_endpoint_id');
    }

    public function scopeActive($query)
    {
        return $query->where('is_active', true)
            ->whereNull('disabled_at');
    }

    public function scopeForWorkspace($query, int $workspaceId)
    {
        return $query->where('workspace_id', $workspaceId);
    }

    public function scopeForEvent($query, string $eventType)
    {
        return $query->where(function ($builder) use ($eventType) {
            $builder->whereJsonContains('events', $eventType)
                ->orWhereJsonContains('events', '*');
        });
    }
}
