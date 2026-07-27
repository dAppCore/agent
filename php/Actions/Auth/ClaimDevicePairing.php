<?php

// SPDX-License-Identifier: EUPL-1.2

declare(strict_types=1);

namespace Core\Mod\Agentic\Actions\Auth;

use Core\Actions\Action;
use Core\Mod\Agentic\Models\AgentApiKey;
use Core\Mod\Agentic\Models\DevicePairing;
use Core\Mod\Agentic\Services\AgentApiKeyService;
use Illuminate\Support\Facades\DB;

/**
 * Exchange a 6-digit pairing code for a freshly minted AgentApiKey.
 *
 * Unauthenticated by design — the short-lived, single-use code is the proof.
 * The returned key carries its one-time plaintext value on ->plainTextKey.
 *
 * @throws InvalidPairingCode when the code is unknown, expired, or already spent
 */
class ClaimDevicePairing
{
    use Action;

    public function handle(string $code): AgentApiKey
    {
        $code = trim($code);

        // Claim the pairing inside a transaction with a row lock so two agents
        // racing the same code cannot both mint a key.
        return DB::transaction(function () use ($code): AgentApiKey {
            $pairing = DevicePairing::query()
                ->where('code', $code)
                ->active()
                ->lockForUpdate()
                ->first();

            if ($pairing === null) {
                throw new InvalidPairingCode;
            }

            $key = app(AgentApiKeyService::class)->create(
                $pairing->workspace_id,
                $pairing->label ?: 'paired-agent-'.$pairing->code,
                $pairing->permissions ?? DevicePairing::DEFAULT_PERMISSIONS,
                $pairing->rate_limit,
                $pairing->keyExpiry(),
            );

            $pairing->markConsumed($key);

            return $key;
        });
    }
}
