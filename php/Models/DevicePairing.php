<?php

// SPDX-License-Identifier: EUPL-1.2

declare(strict_types=1);

namespace Core\Mod\Agentic\Models;

use Illuminate\Database\Eloquent\Builder;
use Illuminate\Database\Eloquent\Model;
use Illuminate\Support\Carbon;

/**
 * Device pairing — a short-lived 6-digit code that bootstraps a fleet node.
 *
 * A logged-in user mints a pairing at app.lthn.ai/device; the agent exchanges
 * the code for an AgentApiKey via POST /v1/agent/auth/login (or /v1/device/pair)
 * without any pre-existing key. The code IS the proof, so it is single-use and
 * expires fast.
 *
 * @property int $id
 * @property string $code
 * @property int $workspace_id
 * @property int|null $user_id
 * @property string|null $label
 * @property array $permissions
 * @property int $rate_limit
 * @property \Carbon\Carbon|null $key_expires_at
 * @property \Carbon\Carbon $expires_at
 * @property \Carbon\Carbon|null $consumed_at
 * @property int|null $agent_api_key_id
 */
class DevicePairing extends Model
{
    protected $table = 'agent_device_pairings';

    /** Minutes a pairing code stays claimable before it expires. */
    public const TTL_MINUTES = 10;

    /**
     * Permissions granted to a worker agent paired through the device flow.
     * Broad enough to be useful out of the box; tighten per-pairing later.
     */
    public const DEFAULT_PERMISSIONS = [
        AgentApiKey::PERM_FLEET_READ,
        AgentApiKey::PERM_FLEET_WRITE,
        AgentApiKey::PERM_SYNC_READ,
        AgentApiKey::PERM_SYNC_WRITE,
        AgentApiKey::PERM_ISSUES_READ,
        AgentApiKey::PERM_ISSUES_WRITE,
        AgentApiKey::PERM_BRAIN_READ,
        AgentApiKey::PERM_BRAIN_WRITE,
        AgentApiKey::PERM_PLANS_READ,
        AgentApiKey::PERM_PLANS_WRITE,
        AgentApiKey::PERM_MESSAGES_READ,
        AgentApiKey::PERM_MESSAGES_WRITE,
    ];

    protected $fillable = [
        'code',
        'workspace_id',
        'user_id',
        'label',
        'permissions',
        'rate_limit',
        'key_expires_at',
        'expires_at',
        'consumed_at',
        'agent_api_key_id',
    ];

    protected $casts = [
        'permissions' => 'array',
        'rate_limit' => 'integer',
        'key_expires_at' => 'datetime',
        'expires_at' => 'datetime',
        'consumed_at' => 'datetime',
    ];

    /**
     * Generate a 6-digit code that is not currently live for any pairing.
     */
    public static function generateCode(): string
    {
        do {
            $code = str_pad((string) random_int(0, 999999), 6, '0', STR_PAD_LEFT);
        } while (static::query()->active()->where('code', $code)->exists());

        return $code;
    }

    /** Pairings that can still be claimed: not consumed, not expired. */
    public function scopeActive(Builder $query): Builder
    {
        return $query->whereNull('consumed_at')
            ->where('expires_at', '>', now());
    }

    public function isClaimable(): bool
    {
        return $this->consumed_at === null && $this->expires_at->isFuture();
    }

    /**
     * Mark this pairing as spent, linking the minted key.
     */
    public function markConsumed(AgentApiKey $key): void
    {
        $this->forceFill([
            'consumed_at' => now(),
            'agent_api_key_id' => $key->id,
        ])->save();
    }

    /** Seconds remaining before the code expires (0 once past). */
    public function secondsRemaining(): int
    {
        return (int) max(0, now()->diffInSeconds($this->expires_at, false));
    }

    /** Resolve the key-expiry to set on the minted AgentApiKey, if any. */
    public function keyExpiry(): ?Carbon
    {
        return $this->key_expires_at;
    }
}
