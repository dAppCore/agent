<?php

declare(strict_types=1);

namespace Core\Mod\Agentic\Controllers\Api;

use Core\Front\Controller;
use Core\Mod\Agentic\Actions\Subscription\DetectCapabilities;
use Core\Mod\Agentic\Actions\Subscription\GetNodeBudget;
use Core\Mod\Agentic\Actions\Subscription\UpdateBudget;
use Illuminate\Http\JsonResponse;
use Illuminate\Http\Request;

class SubscriptionController extends Controller
{
    public function detect(Request $request): JsonResponse
    {
        $validated = $request->validate([
            'api_keys' => 'nullable|array',
        ]);

        $capabilities = DetectCapabilities::run($validated['api_keys'] ?? []);

        return response()->json(['data' => $capabilities]);
    }

    public function budget(Request $request, string $agentId): JsonResponse
    {
        $budget = GetNodeBudget::run((int) $request->attributes->get('workspace_id'), $agentId);

        return response()->json(['data' => $budget]);
    }

    public function updateBudget(Request $request, string $agentId): JsonResponse
    {
        $validated = $request->validate([
            'limits' => 'required|array',
        ]);

        $budget = UpdateBudget::run(
            (int) $request->attributes->get('workspace_id'),
            $agentId,
            $validated['limits'],
        );

        return response()->json(['data' => $budget]);
    }
}
