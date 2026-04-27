{{-- SPDX-License-Identifier: EUPL-1.2 --}}

<div class="space-y-6">
    <div class="grid gap-4 md:grid-cols-2 xl:grid-cols-4">
        <flux:card class="space-y-1">
            <flux:text class="text-xs uppercase tracking-wide text-zinc-500">Fleet Nodes</flux:text>
            <flux:heading size="xl">{{ number_format($this->stats['nodes_total']) }}</flux:heading>
            <flux:text>{{ number_format($this->stats['nodes_online']) }} online, {{ number_format($this->stats['nodes_busy']) }} busy</flux:text>
        </flux:card>

        <flux:card class="space-y-1">
            <flux:text class="text-xs uppercase tracking-wide text-zinc-500">Dispatch Volume</flux:text>
            <flux:heading size="xl">{{ number_format($this->stats['tasks_today']) }}</flux:heading>
            <flux:text>{{ number_format($this->stats['tasks_week']) }} tasks in the last 7 days</flux:text>
        </flux:card>

        <flux:card class="space-y-1">
            <flux:text class="text-xs uppercase tracking-wide text-zinc-500">Repos Touched</flux:text>
            <flux:heading size="xl">{{ number_format($this->stats['repos_touched']) }}</flux:heading>
            <flux:text>{{ number_format($this->stats['findings_total']) }} findings captured</flux:text>
        </flux:card>

        <flux:card class="space-y-1">
            <flux:text class="text-xs uppercase tracking-wide text-zinc-500">Compute Hours</flux:text>
            <flux:heading size="xl">{{ number_format($this->stats['compute_hours']) }}</flux:heading>
            <flux:text>{{ number_format($this->stats['nodes_idle']) }} nodes currently idle</flux:text>
        </flux:card>
    </div>

    <div class="grid gap-6 xl:grid-cols-[minmax(0,2fr)_minmax(24rem,1fr)]">
        <flux:card class="space-y-4">
            <div class="flex items-start justify-between gap-4">
                <div>
                    <flux:heading size="lg">Fleet Overview</flux:heading>
                    <flux:text>Node health, activity state, and dispatch readiness.</flux:text>
                </div>

                <flux:button type="button" variant="ghost" wire:click="refreshOverview">
                    Refresh
                </flux:button>
            </div>

            <div class="grid gap-4 md:grid-cols-2">
                <div class="space-y-2">
                    <flux:text class="text-xs uppercase tracking-wide text-zinc-500">Status Filter</flux:text>
                    <flux:select wire:model.live="statusFilter">
                        <option value="">All statuses</option>
                        <option value="online">Online</option>
                        <option value="busy">Busy</option>
                        <option value="paused">Paused</option>
                        <option value="offline">Offline</option>
                    </flux:select>
                </div>

                <div class="space-y-2">
                    <flux:text class="text-xs uppercase tracking-wide text-zinc-500">Platform Filter</flux:text>
                    <flux:select wire:model.live="platformFilter">
                        <option value="">All platforms</option>
                        @foreach ($this->platforms as $platform)
                            <option value="{{ $platform }}">{{ $platform }}</option>
                        @endforeach
                    </flux:select>
                </div>
            </div>

            <div class="overflow-x-auto rounded-2xl border border-zinc-200 bg-white">
                <table class="min-w-full divide-y divide-zinc-200 text-left text-sm">
                    <thead class="bg-zinc-50 text-xs uppercase tracking-wide text-zinc-500">
                        <tr>
                            <th class="px-4 py-3">Node</th>
                            <th class="px-4 py-3">Status</th>
                            <th class="px-4 py-3">Current Task</th>
                            <th class="px-4 py-3">Budget</th>
                            <th class="px-4 py-3">Heartbeat</th>
                            <th class="px-4 py-3 text-right">Action</th>
                        </tr>
                    </thead>
                    <tbody class="divide-y divide-zinc-100">
                        @forelse ($this->nodes as $node)
                            <tr class="align-top">
                                <td class="px-4 py-3">
                                    <div class="font-medium text-zinc-900">{{ $node['agent_id'] }}</div>
                                    <div class="mt-1 text-xs text-zinc-500">{{ $node['platform'] }}</div>
                                    @if ($node['models'] !== [])
                                        <div class="mt-2 flex flex-wrap gap-1">
                                            @foreach ($node['models'] as $model)
                                                <flux:badge color="zinc">{{ $model }}</flux:badge>
                                            @endforeach
                                        </div>
                                    @endif
                                </td>
                                <td class="px-4 py-3">
                                    <flux:badge color="{{ $this->statusBadgeVariant($node['status']) }}">
                                        {{ strtoupper($node['status']) }}
                                    </flux:badge>
                                </td>
                                <td class="px-4 py-3 text-zinc-700">
                                    {{ $node['current_task_label'] }}
                                </td>
                                <td class="px-4 py-3 text-zinc-700">
                                    {{ $node['compute_budget_label'] }}
                                </td>
                                <td class="px-4 py-3">
                                    <div class="text-zinc-900">{{ $node['last_heartbeat_human'] }}</div>
                                    @if ($node['last_heartbeat_at'])
                                        <div class="mt-1 text-xs text-zinc-500">{{ $node['last_heartbeat_at'] }}</div>
                                    @endif
                                </td>
                                <td class="px-4 py-3 text-right">
                                    <flux:button type="button" variant="primary" wire:click="stageDispatch('{{ $node['agent_id'] }}')">
                                        Dispatch
                                    </flux:button>
                                </td>
                            </tr>
                        @empty
                            <tr>
                                <td colspan="6" class="px-4 py-8 text-center text-sm text-zinc-500">
                                    No fleet nodes match the current filters.
                                </td>
                            </tr>
                        @endforelse
                    </tbody>
                </table>
            </div>
        </flux:card>

        <flux:card class="space-y-4">
            <div>
                <flux:heading size="lg">Dispatch Task</flux:heading>
                <flux:text>Assign work to a specific node without leaving the fleet view.</flux:text>
            </div>

            <form class="space-y-4" wire:submit="dispatchTask">
                <div class="space-y-2">
                    <flux:text class="text-xs uppercase tracking-wide text-zinc-500">Agent</flux:text>
                    <flux:select wire:model="dispatchAgentId">
                        <option value="">Select a node</option>
                        @foreach ($this->nodes as $node)
                            <option value="{{ $node['agent_id'] }}">{{ $node['agent_id'] }} · {{ $node['platform'] }}</option>
                        @endforeach
                    </flux:select>
                    @error('dispatchAgentId') <div class="text-sm text-red-600">{{ $message }}</div> @enderror
                </div>

                <div class="space-y-2">
                    <flux:text class="text-xs uppercase tracking-wide text-zinc-500">Repository</flux:text>
                    <flux:input wire:model="dispatchRepo" placeholder="dAppCore/core-agent" />
                    @error('dispatchRepo') <div class="text-sm text-red-600">{{ $message }}</div> @enderror
                </div>

                <div class="grid gap-4 md:grid-cols-2">
                    <div class="space-y-2">
                        <flux:text class="text-xs uppercase tracking-wide text-zinc-500">Branch</flux:text>
                        <flux:input wire:model="dispatchBranch" placeholder="dev" />
                    </div>

                    <div class="space-y-2">
                        <flux:text class="text-xs uppercase tracking-wide text-zinc-500">Template</flux:text>
                        <flux:input wire:model="dispatchTemplate" placeholder="bugfix" />
                    </div>
                </div>

                <div class="space-y-2">
                    <flux:text class="text-xs uppercase tracking-wide text-zinc-500">Agent Model</flux:text>
                    <flux:input wire:model="dispatchModel" placeholder="gpt-5.5" />
                </div>

                <div class="space-y-2">
                    <flux:text class="text-xs uppercase tracking-wide text-zinc-500">Task</flux:text>
                    <flux:textarea wire:model="dispatchTask" rows="8" placeholder="Describe the work item, constraints, and expected output." />
                    @error('dispatchTask') <div class="text-sm text-red-600">{{ $message }}</div> @enderror
                </div>

                <div class="flex items-center justify-between gap-3">
                    <flux:text>Selected node: {{ $dispatchAgentId !== '' ? $dispatchAgentId : 'none' }}</flux:text>

                    <flux:button type="submit" variant="primary">
                        Dispatch Task
                    </flux:button>
                </div>
            </form>
        </flux:card>
    </div>
</div>
