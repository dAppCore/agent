<?php

// SPDX-License-Identifier: EUPL-1.2

declare(strict_types=1);

namespace Core\Mod\Agentic\Services;

use Core\Mod\Agentic\Data\CreditTransaction;
use Core\Mod\Agentic\Models\CreditEntry;
use Core\Tenant\Models\Workspace;
use Illuminate\Support\Collection;
use Illuminate\Support\Facades\DB;
use InvalidArgumentException;

class CreditService
{
    public function balance(Workspace|int $workspace): array
    {
        $workspaceId = $this->resolveWorkspaceId($workspace);
        $entries = CreditEntry::query()->where('workspace_id', $workspaceId);

        return [
            'workspace_id' => $workspaceId,
            'balance' => (int) (clone $entries)->sum('amount'),
            'total_earned' => (int) (clone $entries)->where('amount', '>', 0)->sum('amount'),
            'total_spent' => (int) abs((int) (clone $entries)->where('amount', '<', 0)->sum('amount')),
            'entries' => (int) (clone $entries)->count(),
        ];
    }

    public function deduct(Workspace|int $workspace, int $amount, string $reason): CreditTransaction
    {
        return $this->record($workspace, -abs($amount), 'manual-deduction', $reason);
    }

    public function refund(Workspace|int $workspace, int $amount, string $reason): CreditTransaction
    {
        return $this->record($workspace, abs($amount), 'manual-refund', $reason);
    }

    /**
     * @return Collection<int, CreditTransaction>
     */
    public function ledger(Workspace|int $workspace): Collection
    {
        $workspaceId = $this->resolveWorkspaceId($workspace);

        return CreditEntry::query()
            ->where('workspace_id', $workspaceId)
            ->latest('id')
            ->get()
            ->map(static fn (CreditEntry $entry): CreditTransaction => CreditTransaction::fromModel($entry));
    }

    private function record(Workspace|int $workspace, int $amount, string $taskType, string $reason): CreditTransaction
    {
        $workspaceId = $this->resolveWorkspaceId($workspace);
        $reason = trim($reason);

        if ($amount === 0) {
            throw new InvalidArgumentException('amount must be greater than zero');
        }

        if ($reason === '') {
            throw new InvalidArgumentException('reason is required');
        }

        $entry = DB::transaction(function () use ($workspaceId, $amount, $taskType, $reason): CreditEntry {
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

        return CreditTransaction::fromModel($entry);
    }

    private function resolveWorkspaceId(Workspace|int $workspace): int
    {
        $workspaceId = $workspace instanceof Workspace ? (int) $workspace->id : (int) $workspace;

        if ($workspaceId <= 0) {
            throw new InvalidArgumentException('workspace_id is required');
        }

        return $workspaceId;
    }
}
