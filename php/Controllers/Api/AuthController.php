<?php

declare(strict_types=1);

namespace Core\Mod\Agentic\Controllers\Api;

use Core\Front\Controller;
use Core\Mod\Agentic\Actions\Auth\ProvisionAgentKey;
use Core\Mod\Agentic\Actions\Auth\RevokeAgentKey;
use Illuminate\Http\JsonResponse;
use Illuminate\Http\Request;

class AuthController extends Controller
{
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
