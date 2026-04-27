<?php

// SPDX-License-Identifier: EUPL-1.2

declare(strict_types=1);

namespace Core\Mod\Agentic\Mod\Api\Models;

use Illuminate\Database\Eloquent\Model;
use Illuminate\Database\Eloquent\Relations\BelongsTo;
use Illuminate\Support\Str;

class WebhookDelivery extends Model
{
    public const STATUS_PENDING = 'pending';

    public const STATUS_QUEUED = 'queued';

    public const STATUS_SUCCESS = 'success';

    public const STATUS_FAILED = 'failed';

    public const STATUS_RETRYING = 'retrying';

    public const STATUS_CANCELLED = 'cancelled';

    public const MAX_ATTEMPTS = 5;

    /**
     * Retry delays keyed by the failed attempt number.
     */
    public const RETRY_DELAYS = [
        1 => 300,
        2 => 900,
        3 => 3600,
        4 => 14400,
        5 => 86400,
    ];

    protected $fillable = [
        'webhook_endpoint_id',
        'event_id',
        'event_type',
        'payload',
        'response_code',
        'response_body',
        'attempt',
        'status',
        'delivered_at',
        'next_retry_at',
    ];

    protected $casts = [
        'payload' => 'array',
        'delivered_at' => 'datetime',
        'next_retry_at' => 'datetime',
    ];

    public static function createForEvent(
        WebhookEndpoint $endpoint,
        string $eventType,
        array $data,
        ?int $workspaceId = null
    ): static {
        return static::query()->create([
            'webhook_endpoint_id' => $endpoint->getKey(),
            'event_id' => 'evt_'.Str::random(24),
            'event_type' => $eventType,
            'payload' => [
                'id' => 'evt_'.Str::random(24),
                'type' => $eventType,
                'created_at' => now()->toIso8601String(),
                'workspace_id' => $workspaceId,
                'data' => $data,
            ],
            'status' => self::STATUS_PENDING,
            'attempt' => 1,
        ]);
    }

    public function markSuccess(int $responseCode, ?string $responseBody = null): void
    {
        $this->forceFill([
            'status' => self::STATUS_SUCCESS,
            'response_code' => $responseCode,
            'response_body' => $responseBody !== null ? Str::limit($responseBody, 10000) : null,
            'delivered_at' => now(),
            'next_retry_at' => null,
        ])->save();

        $this->endpoint?->recordSuccess();
    }

    public function markFailed(int $responseCode, ?string $responseBody = null): void
    {
        $this->endpoint?->recordFailure();

        if ($this->attempt >= self::MAX_ATTEMPTS) {
            $this->forceFill([
                'status' => self::STATUS_FAILED,
                'response_code' => $responseCode,
                'response_body' => $responseBody !== null ? Str::limit($responseBody, 10000) : null,
                'next_retry_at' => null,
            ])->save();

            return;
        }

        $delay = self::RETRY_DELAYS[$this->attempt] ?? end(self::RETRY_DELAYS);

        $this->forceFill([
            'status' => self::STATUS_RETRYING,
            'response_code' => $responseCode,
            'response_body' => $responseBody !== null ? Str::limit($responseBody, 10000) : null,
            'attempt' => $this->attempt + 1,
            'next_retry_at' => now()->addSeconds((int) $delay),
        ])->save();
    }

    public function canRetry(): bool
    {
        return $this->attempt < self::MAX_ATTEMPTS
            && $this->status !== self::STATUS_SUCCESS;
    }

    /**
     * @return array{headers: array<string, string>, body: string}
     */
    public function getDeliveryPayload(?int $timestamp = null): array
    {
        $timestamp ??= time();
        $body = json_encode($this->payload, JSON_THROW_ON_ERROR);

        return [
            'headers' => [
                'Content-Type' => 'application/json',
                'X-Webhook-Id' => $this->event_id,
                'X-Webhook-Event' => $this->event_type,
                'X-Webhook-Timestamp' => (string) $timestamp,
                'X-Webhook-Signature' => $this->endpoint->generateSignature($body, $timestamp),
            ],
            'body' => $body,
        ];
    }

    /**
     * @return array<int, int>
     */
    public static function retrySchedule(): array
    {
        return array_values(self::RETRY_DELAYS);
    }

    public function endpoint(): BelongsTo
    {
        return $this->belongsTo(WebhookEndpoint::class, 'webhook_endpoint_id');
    }

    public function scopeNeedsDelivery($query)
    {
        return $query->where(function ($builder) {
            $builder->where('status', self::STATUS_PENDING)
                ->orWhere(function ($retrying) {
                    $retrying->where('status', self::STATUS_RETRYING)
                        ->where('next_retry_at', '<=', now());
                });
        });
    }
}
