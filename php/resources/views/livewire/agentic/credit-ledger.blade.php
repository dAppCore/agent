{{-- SPDX-License-Identifier: EUPL-1.2 --}}

<div class="space-y-6">
    <flux:card class="space-y-2">
        <flux:heading size="lg">Credit Ledger</flux:heading>
        <flux:text>Balance management, transaction history, and manual credit adjustments for fleet nodes.</flux:text>
    </flux:card>

    <div class="grid gap-4 lg:grid-cols-[minmax(0,20rem)_repeat(3,minmax(0,1fr))]">
        <flux:card class="space-y-2">
            <flux:text class="text-xs uppercase tracking-wide text-zinc-500">Agent</flux:text>
            <flux:select wire:model.live="selectedAgentId">
                <option value="">Select an agent</option>
                @foreach ($this->agents as $agent)
                    <option value="{{ $agent['agent_id'] }}">{{ $agent['agent_id'] }} · {{ $agent['platform'] }}</option>
                @endforeach
            </flux:select>
        </flux:card>

        <flux:card class="space-y-1">
            <flux:text class="text-xs uppercase tracking-wide text-zinc-500">Balance</flux:text>
            <flux:heading size="xl">{{ number_format((int) ($this->balance['balance'] ?? 0)) }}</flux:heading>
            <flux:text>Current balance for {{ $selectedAgentId !== '' ? $selectedAgentId : 'no agent selected' }}</flux:text>
        </flux:card>

        <flux:card class="space-y-1">
            <flux:text class="text-xs uppercase tracking-wide text-zinc-500">Credits Awarded</flux:text>
            <flux:heading size="xl">{{ number_format($this->totals['credits_awarded']) }}</flux:heading>
            <flux:text>Visible positive entries</flux:text>
        </flux:card>

        <flux:card class="space-y-1">
            <flux:text class="text-xs uppercase tracking-wide text-zinc-500">Credits Deducted</flux:text>
            <flux:heading size="xl">{{ number_format($this->totals['credits_deducted']) }}</flux:heading>
            <flux:text>{{ number_format((int) ($this->balance['entries'] ?? 0)) }} total ledger entries</flux:text>
        </flux:card>
    </div>

    <div class="grid gap-6 xl:grid-cols-[minmax(22rem,26rem)_minmax(0,1fr)]">
        <flux:card class="space-y-4">
            <div>
                <flux:heading size="lg">Manual Adjustment</flux:heading>
                <flux:text>Deduct or refund credit for the selected agent.</flux:text>
            </div>

            <div class="space-y-2">
                <flux:text class="text-xs uppercase tracking-wide text-zinc-500">Amount</flux:text>
                <flux:input type="number" min="1" wire:model="adjustmentAmount" />
                @error('adjustmentAmount') <div class="text-sm text-red-600">{{ $message }}</div> @enderror
            </div>

            <div class="space-y-2">
                <flux:text class="text-xs uppercase tracking-wide text-zinc-500">Reason</flux:text>
                <flux:textarea wire:model="adjustmentReason" rows="6" placeholder="Explain why the ledger is being adjusted." />
                @error('adjustmentReason') <div class="text-sm text-red-600">{{ $message }}</div> @enderror
            </div>

            <div class="flex flex-wrap items-center gap-3">
                <flux:button type="button" variant="danger" wire:click="deductCredits">
                    Deduct Credits
                </flux:button>

                <flux:button type="button" variant="primary" wire:click="refundCredits">
                    Refund Credits
                </flux:button>

                <flux:button type="button" variant="ghost" wire:click="refreshLedger">
                    Refresh
                </flux:button>
            </div>
        </flux:card>

        <flux:card class="space-y-4">
            <div class="flex items-start justify-between gap-4">
                <div>
                    <flux:heading size="lg">Transaction Ledger</flux:heading>
                    <flux:text>Recent credit activity for the selected node.</flux:text>
                </div>

                <flux:badge color="zinc">
                    Showing {{ $this->totals['entries_visible'] }} entries
                </flux:badge>
            </div>

            <div class="overflow-x-auto rounded-2xl border border-zinc-200 bg-white">
                <table class="min-w-full divide-y divide-zinc-200 text-left text-sm">
                    <thead class="bg-zinc-50 text-xs uppercase tracking-wide text-zinc-500">
                        <tr>
                            <th class="px-4 py-3">Type</th>
                            <th class="px-4 py-3">Amount</th>
                            <th class="px-4 py-3">Balance After</th>
                            <th class="px-4 py-3">Description</th>
                            <th class="px-4 py-3">Created</th>
                        </tr>
                    </thead>
                    <tbody class="divide-y divide-zinc-100">
                        @forelse ($this->transactions as $transaction)
                            <tr>
                                <td class="px-4 py-3 font-medium text-zinc-900">{{ $transaction['task_type'] }}</td>
                                <td class="px-4 py-3">
                                    <flux:badge color="{{ $this->amountBadgeVariant((int) $transaction['amount']) }}">
                                        {{ (int) $transaction['amount'] > 0 ? '+' : '' }}{{ number_format((int) $transaction['amount']) }}
                                    </flux:badge>
                                </td>
                                <td class="px-4 py-3 text-zinc-700">{{ number_format((int) $transaction['balance_after']) }}</td>
                                <td class="px-4 py-3 text-zinc-700">{{ $transaction['description'] ?: 'No description' }}</td>
                                <td class="px-4 py-3 text-zinc-500">{{ $transaction['created_at'] ?: 'Pending' }}</td>
                            </tr>
                        @empty
                            <tr>
                                <td colspan="5" class="px-4 py-8 text-center text-sm text-zinc-500">
                                    No credit transactions recorded for this agent.
                                </td>
                            </tr>
                        @endforelse
                    </tbody>
                </table>
            </div>
        </flux:card>
    </div>
</div>
