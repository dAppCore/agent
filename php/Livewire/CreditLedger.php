<?php

// SPDX-License-Identifier: EUPL-1.2

declare(strict_types=1);

namespace Core\Mod\Agentic\Livewire;

use Core\Mod\Agentic\Actions\Credits\AwardCredits;
use Core\Mod\Agentic\Actions\Credits\GetBalance;
use Core\Mod\Agentic\Actions\Credits\GetCreditHistory;
use Core\Mod\Agentic\Models\CreditEntry;
use Core\Mod\Agentic\Models\FleetNode;
use Illuminate\Validation\Rule;
use Livewire\Attributes\Computed;
use Livewire\Attributes\Layout;
use Livewire\Attributes\Title;

#[Title('Credit Ledger')]
#[Layout('hub::admin.layouts.app')]
/**
 * Review balances and post manual credit adjustments for workspace agents.
 *
 * @example
 * Livewire::test(CreditLedger::class, ['workspaceId' => 7])->call('refreshLedger');
 */
class CreditLedger extends HubComponent
{
    public int $workspaceId = 0;

    public string $selectedAgentId = '';

    public int $historyLimit = 25;

    public int $adjustmentAmount = 1;

    public string $adjustmentReason = '';

    /**
     * Initialise the ledger with access checks and a selected agent.
     *
     * @example
     * $component->mount(7);
     */
    public function mount(?int $workspaceId = null): void
    {
        $this->checkHadesAccess();
        $this->workspaceId = $workspaceId ?? $this->resolveWorkspaceId();
        $this->syncSelectedAgentId();
    }

    #[Computed]
    /**
     * Return the workspace agents available for ledger inspection.
     *
     * @example
     * $agents = $component->agents();
     */
    public function agents(): array
    {
        if ($this->workspaceId <= 0) {
            return [];
        }

        try {
            return FleetNode::query()
                ->where('workspace_id', $this->workspaceId)
                ->orderBy('agent_id')
                ->get()
                ->map(static fn (FleetNode $node): array => [
                    'id' => $node->id,
                    'agent_id' => $node->agent_id,
                    'platform' => $node->platform,
                    'status' => $node->status,
                ])
                ->values()
                ->all();
        } catch (\Throwable) {
            return [];
        }
    }

    #[Computed]
    /**
     * Return the current balance summary for the selected agent.
     *
     * @example
     * $balance = $component->balance();
     */
    public function balance(): array
    {
        if ($this->workspaceId <= 0 || $this->selectedAgentId === '') {
            return [
                'agent_id' => $this->selectedAgentId,
                'balance' => 0,
                'entries' => 0,
            ];
        }

        try {
            return GetBalance::run($this->workspaceId, $this->selectedAgentId);
        } catch (\Throwable) {
            return [
                'agent_id' => $this->selectedAgentId,
                'balance' => 0,
                'entries' => 0,
            ];
        }
    }

    #[Computed]
    /**
     * Return recent credit transactions for the selected agent.
     *
     * @example
     * $transactions = $component->transactions();
     */
    public function transactions(): array
    {
        if ($this->workspaceId <= 0 || $this->selectedAgentId === '') {
            return [];
        }

        try {
            return GetCreditHistory::run($this->workspaceId, $this->selectedAgentId, $this->historyLimit)
                ->map(static fn (CreditEntry $entry): array => [
                    'id' => $entry->id,
                    'task_type' => $entry->task_type,
                    'amount' => $entry->amount,
                    'balance_after' => $entry->balance_after,
                    'description' => $entry->description,
                    'created_at' => $entry->created_at?->toDateTimeString(),
                ])
                ->values()
                ->all();
        } catch (\Throwable) {
            return [];
        }
    }

    #[Computed]
    /**
     * Summarise the visible transaction totals for the ledger screen.
     *
     * @example
     * $totals = $component->totals();
     */
    public function totals(): array
    {
        $transactions = collect($this->transactions);

        return [
            'credits_awarded' => (int) $transactions->where('amount', '>', 0)->sum('amount'),
            'credits_deducted' => (int) abs($transactions->where('amount', '<', 0)->sum('amount')),
            'entries_visible' => $transactions->count(),
        ];
    }

    /**
     * Add credits to the selected agent and refresh the ledger.
     *
     * @example
     * Livewire::test(CreditLedger::class, ['workspaceId' => 7])->call('refundCredits');
     */
    public function refundCredits(): void
    {
        $this->validateAdjustment();

        AwardCredits::run(
            $this->workspaceId,
            $this->selectedAgentId,
            'manual-refund',
            abs($this->adjustmentAmount),
            null,
            $this->adjustmentReason !== '' ? $this->adjustmentReason : 'Manual refund via admin ledger',
        );

        $this->resetAdjustment();
        $this->refreshLedger();
        $this->toast('Credits Refunded', "Added credit to {$this->selectedAgentId}.", 'success');
    }

    /**
     * Deduct credits from the selected agent and refresh the ledger.
     *
     * @example
     * Livewire::test(CreditLedger::class, ['workspaceId' => 7])->call('deductCredits');
     */
    public function deductCredits(): void
    {
        $this->validateAdjustment();

        AwardCredits::run(
            $this->workspaceId,
            $this->selectedAgentId,
            'manual-deduction',
            -abs($this->adjustmentAmount),
            null,
            $this->adjustmentReason !== '' ? $this->adjustmentReason : 'Manual deduction via admin ledger',
        );

        $this->resetAdjustment();
        $this->refreshLedger();
        $this->toast('Credits Deducted', "Deducted credit from {$this->selectedAgentId}.", 'warning');
    }

    /**
     * Refresh the computed ledger data and resynchronise the selected agent.
     *
     * @example
     * Livewire::test(CreditLedger::class, ['workspaceId' => 7])->call('refreshLedger');
     */
    public function refreshLedger(): void
    {
        unset($this->agents, $this->balance, $this->transactions, $this->totals);
        $this->syncSelectedAgentId();
        $this->dispatch('notify', message: 'Credit ledger refreshed');
    }

    /**
     * Map a credit amount to the badge variant used in the ledger UI.
     *
     * @example
     * $variant = $component->amountBadgeVariant(-5);
     */
    public function amountBadgeVariant(int $amount): string
    {
        return $amount >= 0 ? 'success' : 'danger';
    }

    /**
     * Validate the current manual adjustment form values.
     *
     * @example
     * $this->validateAdjustment();
     */
    private function validateAdjustment(): void
    {
        $this->validate([
            'workspaceId' => 'required|integer|min:1',
            'selectedAgentId' => [
                'required',
                'string',
                'max:255',
                Rule::exists(FleetNode::class, 'agent_id')->where(
                    fn ($query) => $query->where('workspace_id', $this->workspaceId),
                ),
            ],
            'adjustmentAmount' => 'required|integer|min:1|max:100000',
            'adjustmentReason' => 'nullable|string|max:1000',
        ]);
    }

    /**
     * Reset the manual adjustment form back to its default values.
     *
     * @example
     * $this->resetAdjustment();
     */
    private function resetAdjustment(): void
    {
        $this->adjustmentAmount = 1;
        $this->adjustmentReason = '';
    }

    /**
     * Keep the selected agent aligned with the current workspace agent list.
     *
     * @example
     * $this->syncSelectedAgentId();
     */
    private function syncSelectedAgentId(): void
    {
        if ($this->selectedAgentId !== '' && collect($this->agents)->contains('agent_id', $this->selectedAgentId)) {
            return;
        }

        $this->selectedAgentId = (string) (collect($this->agents)->first()['agent_id'] ?? '');
    }

    /**
     * Resolve the Blade template used by the credit ledger screen.
     *
     * @example
     * $path = $this->viewPath();
     */
    protected function viewPath(): string
    {
        return __DIR__.'/../resources/views/livewire/agentic/credit-ledger.blade.php';
    }
}
