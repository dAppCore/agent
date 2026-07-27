<?php

// SPDX-License-Identifier: EUPL-1.2

declare(strict_types=1);

namespace Core\Mod\Agentic\Actions\Auth;

use Core\Actions\Action;
use Core\Mod\Agentic\Models\DevicePairing;
use Illuminate\Support\Carbon;

/**
 * Mint a short-lived device-pairing code for a workspace.
 *
 * Called from the logged-in app.lthn.ai/device screen. The returned pairing
 * carries the 6-digit code the operator types into `core-agent login`.
 */
class CreateDevicePairing
{
    use Action;

    /**
     * @param  array<string>|null  $permissions  null falls back to the worker default set
     * @param  int|null  $keyTtlDays  optional expiry for the minted key; null = never expires
     *
     * @throws \InvalidArgumentException
     */
    public function handle(
        int $workspaceId,
        ?int $userId = null,
        ?string $label = null,
        ?array $permissions = null,
        int $rateLimit = 100,
        ?int $keyTtlDays = null,
    ): DevicePairing {
        if ($workspaceId <= 0) {
            throw new \InvalidArgumentException('workspace_id is required');
        }

        return DevicePairing::create([
            'code' => DevicePairing::generateCode(),
            'workspace_id' => $workspaceId,
            'user_id' => $userId,
            'label' => $label,
            'permissions' => $permissions ?? DevicePairing::DEFAULT_PERMISSIONS,
            'rate_limit' => $rateLimit,
            'key_expires_at' => $keyTtlDays !== null ? Carbon::now()->addDays($keyTtlDays) : null,
            'expires_at' => Carbon::now()->addMinutes(DevicePairing::TTL_MINUTES),
        ]);
    }
}
