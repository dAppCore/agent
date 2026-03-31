<?php

declare(strict_types=1);

namespace Core\Mod\Agentic\Controllers\Api;

use Core\Front\Controller;
use Core\Mod\Agentic\Actions\Credits\AwardCredits;
use Core\Mod\Agentic\Actions\Credits\GetBalance;
use Core\Mod\Agentic\Actions\Credits\GetCreditHistory;
use Core\Mod\Agentic\Models\CreditEntry;
use Illuminate\Http\JsonResponse;
use Illuminate\Http\Request;

class CreditsController extends Controller
{
    public function award(Request $request): JsonResponse
    {
        $validated = $request->validate([
            'agent_id' => 'required|string|max:255',
            'task_type' => 'required|string|max:255',
            'amount' => 'required|integer|not_in:0',
            'fleet_node_id' => 'nullable|integer',
            'description' => 'nullable|string|max:1000',
        ]);

        $entry = AwardCredits::run(
            (int) $request->attributes->get('workspace_id'),
            $validated['agent_id'],
            $validated['task_type'],
            (int) $validated['amount'],
            isset($validated['fleet_node_id']) ? (int) $validated['fleet_node_id'] : null,
            $validated['description'] ?? null,
        );

        return response()->json(['data' => $this->formatEntry($entry)], 201);
    }

    public function balance(Request $request, string $agentId): JsonResponse
    {
        $balance = GetBalance::run((int) $request->attributes->get('workspace_id'), $agentId);

        return response()->json(['data' => $balance]);
    }

    public function history(Request $request, string $agentId): JsonResponse
    {
        $limit = (int) $request->query('limit', 50);
        $entries = GetCreditHistory::run((int) $request->attributes->get('workspace_id'), $agentId, $limit);

        return response()->json([
            'data' => $entries->map(fn (CreditEntry $entry) => $this->formatEntry($entry))->values()->all(),
            'total' => $entries->count(),
        ]);
    }

    /**
     * @return array<string, mixed>
     */
    private function formatEntry(CreditEntry $entry): array
    {
        return [
            'id' => $entry->id,
            'task_type' => $entry->task_type,
            'amount' => $entry->amount,
            'balance_after' => $entry->balance_after,
            'description' => $entry->description,
            'created_at' => $entry->created_at?->toIso8601String(),
        ];
    }
}
