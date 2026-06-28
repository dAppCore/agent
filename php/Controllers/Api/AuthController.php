<?php

declare(strict_types=1);

namespace Core\Mod\Agentic\Controllers\Api;

use Core\Front\Controller;
use Core\Mod\Agentic\Actions\Auth\ClaimDevicePairing;
use Core\Mod\Agentic\Actions\Auth\InvalidPairingCode;
use Core\Mod\Agentic\Actions\Auth\ProvisionAgentKey;
use Core\Mod\Agentic\Actions\Auth\RevokeAgentKey;
use Core\Mod\Agentic\Models\AgentApiKey;
use Illuminate\Http\JsonResponse;
use Illuminate\Http\Request;

class AuthController extends Controller
{
    /**
     * Exchange a 6-digit device-pairing code for an AgentApiKey.
     *
     * Unauthenticated — the short-lived, single-use code is the proof. Mirrors
     * the Go client's AuthLoginOutput: { data: { key: { …, key: "ak_…" } } }.
     */
    public function login(Request $request): JsonResponse
    {
        $validated = $request->validate([
            'code' => ['required', 'string', 'regex:/^[0-9]{6}$/'],
        ]);

        try {
            $key = ClaimDevicePairing::run($validated['code']);
        } catch (InvalidPairingCode $e) {
            return response()->json([
                'error' => 'invalid_pairing_code',
                'message' => $e->getMessage(),
            ], 422);
        }

        return response()->json([
            'data' => ['key' => $this->keyResource($key)],
        ]);
    }

    /**
     * Fleet-mode device pairing — same exchange, the shape the Go fleet client
     * unwraps: { data: { agent_api_key: "ak_…", agent_id, expires_at } }.
     */
    public function pair(Request $request): JsonResponse
    {
        $validated = $request->validate([
            'code' => ['required', 'string', 'regex:/^[0-9]{6}$/'],
        ]);

        try {
            $key = ClaimDevicePairing::run($validated['code']);
        } catch (InvalidPairingCode $e) {
            return response()->json([
                'error' => 'invalid_pairing_code',
                'message' => $e->getMessage(),
            ], 422);
        }

        return response()->json([
            'data' => [
                'agent_api_key' => $key->plainTextKey,
                'agent_id' => $key->name,
                'expires_at' => $key->expires_at?->toIso8601String(),
            ],
        ]);
    }

    /**
     * Serialise a freshly minted key for the Go AgentApiKey struct, carrying
     * the one-time plaintext value the caller must persist.
     *
     * @return array<string, mixed>
     */
    private function keyResource(AgentApiKey $key): array
    {
        return [
            'id' => $key->id,
            'workspace_id' => $key->workspace_id,
            'name' => $key->name,
            'key' => $key->plainTextKey,
            'permissions' => $key->permissions ?? [],
            'rate_limit' => $key->rate_limit,
            'expires_at' => $key->expires_at?->toIso8601String(),
            'created_at' => $key->created_at?->toIso8601String(),
        ];
    }

    public function provision(Request $request): JsonResponse
    {
        $validated = $request->validate([
            'workspace_id' => 'required|integer',
            'oauth_user_id' => 'required|string|max:255',
            'name' => 'nullable|string|max:255',
            'permissions' => 'nullable|array',
            'permissions.*' => 'string',
            'rate_limit' => 'nullable|integer|min:1',
            'expires_at' => 'nullable|date',
        ]);

        $key = ProvisionAgentKey::run(
            (int) $validated['workspace_id'],
            $validated['oauth_user_id'],
            $validated['name'] ?? null,
            $validated['permissions'] ?? [],
            (int) ($validated['rate_limit'] ?? 100),
            $validated['expires_at'] ?? null,
        );

        return response()->json([
            'data' => [
                'id' => $key->id,
                'name' => $key->name,
                'plain_text_key' => $key->plainTextKey,
                'permissions' => $key->permissions ?? [],
                'rate_limit' => $key->rate_limit,
            ],
        ], 201);
    }

    public function revoke(Request $request, int $keyId): JsonResponse
    {
        RevokeAgentKey::run((int) $request->attributes->get('workspace_id'), $keyId);

        return response()->json([
            'data' => [
                'key_id' => $keyId,
                'revoked' => true,
            ],
        ]);
    }
}
