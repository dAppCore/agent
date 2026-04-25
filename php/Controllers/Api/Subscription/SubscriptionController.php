<?php

// SPDX-License-Identifier: EUPL-1.2

declare(strict_types=1);

namespace Core\Mod\Agentic\Controllers\Api\Subscription;

use Core\Front\Controller;
use Core\Mod\Agentic\Actions\Subscription\DetectCapabilities;
use Core\Mod\Agentic\Actions\Subscription\GetNodeBudget;
use Core\Mod\Agentic\Actions\Subscription\UpdateBudget;
use Core\Mod\Agentic\Models\CreditEntry;
use Core\Mod\Agentic\Services\CreditService;
use Core\Mod\Agentic\Services\SessionService;
use Illuminate\Http\JsonResponse;
use Illuminate\Http\Request;

class SubscriptionController extends Controller
{
    public function status(Request $request): JsonResponse
    {
        $validated = $request->validate([
            'agent_id' => 'nullable|string|max:255',
            'api_keys' => 'nullable|array',
            'api_keys.*' => 'string',
        ]);

        $workspaceId = (int) $request->attributes->get('workspace_id');
        $capabilities = DetectCapabilities::run($validated['api_keys'] ?? []);
        $credits = $this->workspaceBalance($workspaceId);
        $agentId = trim((string) ($validated['agent_id'] ?? ''));

        return response()->json([
            'data' => [
                'workspace_id' => $workspaceId,
                'status' => ! empty($capabilities['available']) || (($credits['balance'] ?? 0) > 0) ? 'active' : 'inactive',
                'providers' => $capabilities['providers'] ?? [],
                'available' => $capabilities['available'] ?? [],
                'credits' => $credits,
                'budget' => $agentId !== '' ? GetNodeBudget::run($workspaceId, $agentId) : null,
            ],
        ]);
    }

    public function upgrade(Request $request): JsonResponse
    {
        $validated = $request->validate([
            'agent_id' => 'required|string|max:255',
            'limits' => 'required|array',
            'session_id' => 'nullable|string|max:255',
        ]);

        $budget = UpdateBudget::run(
            (int) $request->attributes->get('workspace_id'),
            $validated['agent_id'],
            $validated['limits'],
        );

        $this->emitSessionEvent(
            $validated['session_id'] ?? null,
            'subscription.upgraded',
            ['agent_id' => $validated['agent_id'], 'limits' => $validated['limits']],
        );

        return response()->json([
            'data' => [
                'agent_id' => $validated['agent_id'],
                'status' => 'upgraded',
                'budget' => $budget,
            ],
        ]);
    }

    public function cancel(Request $request): JsonResponse
    {
        $validated = $request->validate([
            'agent_id' => 'required|string|max:255',
            'session_id' => 'nullable|string|max:255',
        ]);

        $budget = UpdateBudget::run(
            (int) $request->attributes->get('workspace_id'),
            $validated['agent_id'],
            [
                'cancelled' => true,
                'cancelled_at' => now()->toIso8601String(),
                'max_daily_hours' => 0,
            ],
        );

        $this->emitSessionEvent(
            $validated['session_id'] ?? null,
            'subscription.cancelled',
            ['agent_id' => $validated['agent_id']],
        );

        return response()->json([
            'data' => [
                'agent_id' => $validated['agent_id'],
                'status' => 'cancelled',
                'budget' => $budget,
            ],
        ]);
    }

    /**
     * @return array<string, mixed>
     */
    private function workspaceBalance(int $workspaceId): array
    {
        $service = $this->resolveCreditService();

        if ($service !== null && method_exists($service, 'balance')) {
            return (array) $service->balance($workspaceId);
        }

        $entries = CreditEntry::query()->where('workspace_id', $workspaceId);

        return [
            'workspace_id' => $workspaceId,
            'balance' => (int) (clone $entries)->sum('amount'),
            'total_earned' => (int) (clone $entries)->where('amount', '>', 0)->sum('amount'),
            'total_spent' => (int) abs((int) (clone $entries)->where('amount', '<', 0)->sum('amount')),
            'entries' => (int) (clone $entries)->count(),
        ];
    }

    /**
     * @param  array<string, mixed>  $data
     */
    private function emitSessionEvent(?string $sessionId, string $event, array $data): void
    {
        if ($sessionId === null || trim($sessionId) === '') {
            return;
        }

        $service = $this->resolveSessionService();

        if ($service === null || ! method_exists($service, 'emit')) {
            return;
        }

        $service->emit($sessionId, [
            'event' => $event,
            'data' => $data,
        ]);
    }

    private function resolveCreditService(): ?object
    {
        if (! class_exists(CreditService::class)) {
            return null;
        }

        $service = app(CreditService::class);

        return is_object($service) ? $service : null;
    }

    private function resolveSessionService(): ?object
    {
        if (! class_exists(SessionService::class)) {
            return null;
        }

        $service = app(SessionService::class);

        return is_object($service) ? $service : null;
    }
}
