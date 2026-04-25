<?php

// SPDX-License-Identifier: EUPL-1.2

declare(strict_types=1);

namespace Core\Mod\Agentic\Controllers\Api\Credits;

use Core\Front\Controller;
use Core\Mod\Agentic\Models\CreditEntry;
use Core\Mod\Agentic\Services\CreditService;
use Illuminate\Http\JsonResponse;
use Illuminate\Http\Request;
use Illuminate\Support\Facades\DB;

class CreditsController extends Controller
{
    public function balance(Request $request): JsonResponse
    {
        $workspaceId = (int) $request->attributes->get('workspace_id');
        $service = $this->resolveCreditService();

        $payload = $service !== null && method_exists($service, 'balance')
            ? (array) $service->balance($workspaceId)
            : $this->fallbackBalance($workspaceId);

        return response()->json(['data' => $payload]);
    }

    public function deduct(Request $request): JsonResponse
    {
        $validated = $request->validate([
            'amount' => 'required|integer|min:1',
            'reason' => 'required|string|max:1000',
        ]);

        $workspaceId = (int) $request->attributes->get('workspace_id');
        $service = $this->resolveCreditService();

        $entry = $service !== null && method_exists($service, 'deduct')
            ? $service->deduct($workspaceId, (int) $validated['amount'], $validated['reason'])
            : $this->recordTransaction($workspaceId, -abs((int) $validated['amount']), 'manual-deduction', $validated['reason']);

        return response()->json(['data' => $this->formatEntry($entry)], 201);
    }

    public function refund(Request $request): JsonResponse
    {
        $validated = $request->validate([
            'amount' => 'required|integer|min:1',
            'reason' => 'required|string|max:1000',
        ]);

        $workspaceId = (int) $request->attributes->get('workspace_id');
        $service = $this->resolveCreditService();

        $entry = $service !== null && method_exists($service, 'refund')
            ? $service->refund($workspaceId, (int) $validated['amount'], $validated['reason'])
            : $this->recordTransaction($workspaceId, abs((int) $validated['amount']), 'manual-refund', $validated['reason']);

        return response()->json(['data' => $this->formatEntry($entry)], 201);
    }

    public function ledger(Request $request): JsonResponse
    {
        $validated = $request->validate([
            'limit' => 'nullable|integer|min:1|max:500',
        ]);

        $workspaceId = (int) $request->attributes->get('workspace_id');
        $limit = (int) ($validated['limit'] ?? 50);
        $service = $this->resolveCreditService();

        $entries = [];

        if ($service !== null && method_exists($service, 'ledger')) {
            foreach ($service->ledger($workspaceId) as $entry) {
                $entries[] = $this->formatEntry($entry);

                if (count($entries) >= $limit) {
                    break;
                }
            }
        } else {
            foreach (CreditEntry::query()->where('workspace_id', $workspaceId)->latest('id')->limit($limit)->get() as $entry) {
                $entries[] = $this->formatEntry($entry);
            }
        }

        return response()->json([
            'data' => $entries,
            'total' => count($entries),
        ]);
    }

    /**
     * @return array<string, mixed>
     */
    private function fallbackBalance(int $workspaceId): array
    {
        $entries = CreditEntry::query()->where('workspace_id', $workspaceId);

        return [
            'workspace_id' => $workspaceId,
            'balance' => (int) (clone $entries)->sum('amount'),
            'total_earned' => (int) (clone $entries)->where('amount', '>', 0)->sum('amount'),
            'total_spent' => (int) abs((int) (clone $entries)->where('amount', '<', 0)->sum('amount')),
            'entries' => (int) (clone $entries)->count(),
        ];
    }

    private function recordTransaction(int $workspaceId, int $amount, string $taskType, string $reason): CreditEntry
    {
        return DB::transaction(function () use ($workspaceId, $amount, $taskType, $reason): CreditEntry {
            $previousBalance = (int) CreditEntry::query()
                ->where('workspace_id', $workspaceId)
                ->lockForUpdate()
                ->latest('id')
                ->value('balance_after');

            return CreditEntry::query()->create([
                'workspace_id' => $workspaceId,
                'fleet_node_id' => null,
                'task_type' => $taskType,
                'amount' => $amount,
                'balance_after' => $previousBalance + $amount,
                'description' => $reason,
            ]);
        });
    }

    /**
     * @return array<string, mixed>
     */
    private function formatEntry(object|array $entry): array
    {
        if (is_array($entry)) {
            return [
                'id' => isset($entry['id']) ? (int) $entry['id'] : null,
                'workspace_id' => isset($entry['workspace_id']) ? (int) $entry['workspace_id'] : null,
                'fleet_node_id' => isset($entry['fleet_node_id']) ? (int) $entry['fleet_node_id'] : null,
                'task_type' => (string) ($entry['task_type'] ?? ''),
                'amount' => (int) ($entry['amount'] ?? 0),
                'balance_after' => (int) ($entry['balance_after'] ?? 0),
                'description' => isset($entry['description']) ? (string) $entry['description'] : null,
                'created_at' => isset($entry['created_at']) ? (string) $entry['created_at'] : null,
            ];
        }

        $createdAt = $entry->created_at ?? null;

        return [
            'id' => isset($entry->id) ? (int) $entry->id : null,
            'workspace_id' => isset($entry->workspace_id) ? (int) $entry->workspace_id : null,
            'fleet_node_id' => isset($entry->fleet_node_id) ? (int) $entry->fleet_node_id : null,
            'task_type' => (string) ($entry->task_type ?? ''),
            'amount' => (int) ($entry->amount ?? 0),
            'balance_after' => (int) ($entry->balance_after ?? 0),
            'description' => isset($entry->description) ? (string) $entry->description : null,
            'created_at' => is_object($createdAt) && method_exists($createdAt, 'toIso8601String')
                ? $createdAt->toIso8601String()
                : ($createdAt !== null ? (string) $createdAt : null),
        ];
    }

    private function resolveCreditService(): ?object
    {
        if (! class_exists(CreditService::class)) {
            return null;
        }

        $service = app(CreditService::class);

        return is_object($service) ? $service : null;
    }
}
