<?php

// SPDX-License-Identifier: EUPL-1.2

declare(strict_types=1);

namespace Core\Mod\Agentic\Livewire;

use Core\Mod\Agentic\Actions\Credits\AwardCredits;
use Core\Mod\Agentic\Actions\Credits\GetBalance;
use Core\Mod\Agentic\Actions\Credits\GetCreditHistory;
use Core\Mod\Agentic\Models\CreditEntry;
use Core\Mod\Agentic\Models\FleetNode;
use Core\Tenant\Models\Workspace;
use Flux\Flux;
use Illuminate\Contracts\View\View;
use Livewire\Attributes\Computed;
use Livewire\Attributes\Layout;
use Livewire\Attributes\Title;
use Livewire\Component;

#[Title('Credit Ledger')]
#[Layout('hub::admin.layouts.app')]
class CreditLedger extends Component
{
    public int $workspaceId = 0;

    public string $selectedAgentId = '';

    public int $historyLimit = 25;

    public int $adjustmentAmount = 1;

    public string $adjustmentReason = '';

    public function mount(?int $workspaceId = null): void
    {
        $this->checkHadesAccess();
        $this->workspaceId = $workspaceId ?? $this->resolveWorkspaceId();
        $this->syncSelectedAgentId();
    }

    #[Computed]
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
    public function totals(): array
    {
        $transactions = collect($this->transactions);

        return [
            'credits_awarded' => (int) $transactions->where('amount', '>', 0)->sum('amount'),
            'credits_deducted' => (int) abs($transactions->where('amount', '<', 0)->sum('amount')),
            'entries_visible' => $transactions->count(),
        ];
    }

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

    public function refreshLedger(): void
    {
        unset($this->agents, $this->balance, $this->transactions, $this->totals);
        $this->syncSelectedAgentId();
        $this->dispatch('notify', message: 'Credit ledger refreshed');
    }

    public function amountBadgeVariant(int $amount): string
    {
        return $amount >= 0 ? 'success' : 'danger';
    }

    public function render(): View
    {
        return view()->file($this->viewPath());
    }

    private function validateAdjustment(): void
    {
        $this->validate([
            'workspaceId' => 'required|integer|min:1',
            'selectedAgentId' => 'required|string|max:255',
            'adjustmentAmount' => 'required|integer|min:1|max:100000',
            'adjustmentReason' => 'nullable|string|max:1000',
        ]);
    }

    private function resetAdjustment(): void
    {
        $this->adjustmentAmount = 1;
        $this->adjustmentReason = '';
    }

    private function syncSelectedAgentId(): void
    {
        if ($this->selectedAgentId !== '' && collect($this->agents)->contains('agent_id', $this->selectedAgentId)) {
            return;
        }

        $this->selectedAgentId = (string) (collect($this->agents)->first()['agent_id'] ?? '');
    }

    private function resolveWorkspaceId(): int
    {
        try {
            return (int) (Workspace::query()->value('id') ?? 0);
        } catch (\Throwable) {
            return 0;
        }
    }

    private function checkHadesAccess(): void
    {
        if (! auth()->user()?->isHades()) {
            abort(403, 'Hades access required');
        }
    }

    private function toast(string $heading, string $text, string $variant): void
    {
        if (class_exists(Flux::class)) {
            Flux::toast(
                heading: $heading,
                text: $text,
                variant: $variant,
            );

            return;
        }

        $this->dispatch('notify', message: $text, variant: $variant);
    }

    private function viewPath(): string
    {
        return __DIR__.'/../../resources/views/livewire/agentic/credit-ledger.blade.php';
    }
}
