<?php

// SPDX-License-Identifier: EUPL-1.2

declare(strict_types=1);

namespace Core\Mod\Agentic\Mod\Api\Services;

use Illuminate\Support\Str;

class WebhookSignature
{
    public const DEFAULT_TOLERANCE = 300;

    public function generateSecret(): string
    {
        return Str::random(64);
    }

    public function sign(string $payload, string $secret, int $timestamp): string
    {
        return hash_hmac('sha256', $timestamp.'.'.$payload, $secret);
    }

    public function verify(
        string $payload,
        string $signature,
        string $secret,
        int $timestamp,
        int $tolerance = self::DEFAULT_TOLERANCE
    ): bool {
        if (! $this->isTimestampValid($timestamp, $tolerance)) {
            return false;
        }

        return hash_equals($this->sign($payload, $secret, $timestamp), $signature);
    }

    public function verifySignatureOnly(
        string $payload,
        string $signature,
        string $secret,
        int $timestamp
    ): bool {
        return hash_equals($this->sign($payload, $secret, $timestamp), $signature);
    }

    public function isTimestampValid(int $timestamp, int $tolerance = self::DEFAULT_TOLERANCE): bool
    {
        return abs(time() - $timestamp) <= $tolerance;
    }
}
