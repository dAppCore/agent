<div>
    <core:heading size="xl" class="mb-2">{{ __('agentic::agentic.plans.title') }}</core:heading>
    <core:subheading class="mb-6">{{ __('agentic::agentic.plans.subtitle') }}</core:subheading>

    {{-- Filters --}}
    <core:card class="p-4 mb-6">
        <div class="flex flex-col md:flex-row gap-4">
            <div class="flex-1">
                <core:input
                    wire:model.live.debounce.300ms="search"
                    :placeholder="__('agentic::agentic.plans.search_placeholder')"
                    icon="magnifying-glass"
                />
            </div>
            <div class="w-full md:w-48">
                <core:select wire:model.live="status">
                    <option value="">{{ __('agentic::agentic.filters.all_statuses') }}</option>
                    @foreach($this->statusOptions as $value => $label)
                        <option value="{{ $value }}">{{ $label }}</option>
                    @endforeach
                </core:select>
            </div>
            <div class="w-full md:w-48">
                <core:select wire:model.live="workspace">
                    <option value="">{{ __('agentic::agentic.filters.all_workspaces') }}</option>
                    @foreach($this->workspaces as $ws)
                        <option value="{{ $ws->id }}">{{ $ws->name }}</option>
                    @endforeach
                </core:select>
            </div>
            @if($search || $status || $workspace)
                <core:button wire:click="clearFilters" variant="ghost" icon="x-mark">
                    {{ __('agentic::agentic.actions.clear') }}
                </core:button>
            @endif
        </div>
    </core:card>

    {{-- Plans Table --}}
    <core:card>
        @if($this->plans->count() > 0)
            <div class="overflow-x-auto">
                <table class="w-full">
                    <thead>
                        <tr class="border-b border-zinc-200 dark:border-zinc-700">
                            <th class="text-left p-4 font-medium text-zinc-600 dark:text-zinc-300">{{ __('agentic::agentic.table.plan') }}</th>
                            <th class="text-left p-4 font-medium text-zinc-600 dark:text-zinc-300">{{ __('agentic::agentic.table.workspace') }}</th>
                            <th class="text-left p-4 font-medium text-zinc-600 dark:text-zinc-300">{{ __('agentic::agentic.table.status') }}</th>
                            <th class="text-left p-4 font-medium text-zinc-600 dark:text-zinc-300">{{ __('agentic::agentic.table.progress') }}</th>
                            <th class="text-left p-4 font-medium text-zinc-600 dark:text-zinc-300">{{ __('agentic::agentic.table.sessions') }}</th>
                            <th class="text-left p-4 font-medium text-zinc-600 dark:text-zinc-300">{{ __('agentic::agentic.table.last_activity') }}</th>
                            <th class="text-right p-4 font-medium text-zinc-600 dark:text-zinc-300">{{ __('agentic::agentic.table.actions') }}</th>
                        </tr>
                    </thead>
                    <tbody class="divide-y divide-zinc-200 dark:divide-zinc-700">
                        @foreach($this->plans as $plan)
                            @php
                                $progress = $plan->getProgress();
                                $hasBlockedPhase = $plan->agentPhases->contains('status', 'blocked');
                            @endphp
                            <tr class="hover:bg-zinc-50 dark:hover:bg-zinc-800/50">
                                <td class="p-4">
                                    <a href="{{ route('hub.agents.plans.show', $plan->slug) }}" wire:navigate class="block">
                                        <core:text class="font-medium hover:text-violet-600 dark:hover:text-violet-400">{{ $plan->title }}</core:text>
                                        <core:text size="sm" class="text-zinc-500 truncate max-w-xs">{{ $plan->slug }}</core:text>
                                    </a>
                                </td>
                                <td class="p-4">
                                    <core:text size="sm">{{ $plan->workspace?->name ?? 'N/A' }}</core:text>
                                </td>
                                <td class="p-4">
                                    <div class="flex items-center gap-2">
                                        <span class="inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium
                                            @if($plan->status === 'draft') bg-zinc-100 text-zinc-700 dark:bg-zinc-700 dark:text-zinc-300
                                            @elseif($plan->status === 'active') bg-blue-100 text-blue-700 dark:bg-blue-900/50 dark:text-blue-300
                                            @elseif($plan->status === 'completed') bg-green-100 text-green-700 dark:bg-green-900/50 dark:text-green-300
                                            @else bg-amber-100 text-amber-700 dark:bg-amber-900/50 dark:text-amber-300
                                            @endif">
                                            {{ ucfirst($plan->status) }}
                                        </span>
                                        @if($hasBlockedPhase)
                                            <span class="inline-flex items-center px-2 py-0.5 rounded-full text-xs font-medium bg-red-100 text-red-700 dark:bg-red-900/50 dark:text-red-300">
                                                {{ __('agentic::agentic.status.blocked') }}
                                            </span>
                                        @endif
                                    </div>
                                </td>
                                <td class="p-4">
                                    <div class="flex items-center gap-2">
                                        <div class="w-24 bg-zinc-200 dark:bg-zinc-700 rounded-full h-2">
                                            <div class="bg-violet-500 h-2 rounded-full transition-all" style="width: {{ $progress['percentage'] }}%"></div>
                                        </div>
                                        <core:text size="sm" class="text-zinc-500">{{ $progress['percentage'] }}%</core:text>
                                    </div>
                                    <core:text size="xs" class="text-zinc-400 mt-1">{{ $progress['completed'] }}/{{ $progress['total'] }} phases</core:text>
                                </td>
                                <td class="p-4">
                                    <core:text size="sm">{{ $plan->sessions_count }}</core:text>
                                </td>
                                <td class="p-4">
                                    <core:text size="sm" class="text-zinc-500">{{ $plan->updated_at->diffForHumans() }}</core:text>
                                </td>
                                <td class="p-4">
                                    <div class="flex items-center justify-end gap-2">
                                        <a href="{{ route('hub.agents.plans.show', $plan->slug) }}" wire:navigate>
                                            <core:button variant="ghost" size="sm" icon="eye">{{ __('agentic::agentic.actions.view') }}</core:button>
                                        </a>
                                        <core:dropdown>
                                            <core:button variant="ghost" size="sm" icon="ellipsis-vertical" />
                                            <core:menu>
                                                @if($plan->status === 'draft')
                                                    <core:menu.item wire:click="activate({{ $plan->id }})" icon="play">{{ __('agentic::agentic.actions.activate') }}</core:menu.item>
                                                @endif
                                                @if($plan->status === 'active')
                                                    <core:menu.item wire:click="complete({{ $plan->id }})" icon="check">{{ __('agentic::agentic.actions.complete') }}</core:menu.item>
                                                @endif
                                                @if($plan->status !== 'archived')
                                                    <core:menu.item wire:click="archive({{ $plan->id }})" icon="archive-box">{{ __('agentic::agentic.actions.archive') }}</core:menu.item>
                                                @endif
                                                <core:menu.separator />
                                                <core:menu.item wire:click="delete({{ $plan->id }})" wire:confirm="{{ __('agentic::agentic.confirm.delete_plan') }}" icon="trash" variant="danger">{{ __('agentic::agentic.actions.delete') }}</core:menu.item>
                                            </core:menu>
                                        </core:dropdown>
                                    </div>
                                </td>
                            </tr>
                        @endforeach
                    </tbody>
                </table>
            </div>

            {{-- Pagination --}}
            <div class="p-4 border-t border-zinc-200 dark:border-zinc-700">
                {{ $this->plans->links() }}
            </div>
        @else
            <div class="flex flex-col items-center py-12 text-center">
                <core:icon name="clipboard-document-list" class="w-16 h-16 text-zinc-300 dark:text-zinc-600 mb-4" />
                <core:heading size="lg" class="text-zinc-600 dark:text-zinc-400">{{ __('agentic::agentic.empty.no_plans') }}</core:heading>
                <core:text class="text-zinc-500 mt-2">
                    @if($search || $status || $workspace)
                        {{ __('agentic::agentic.empty.filter_hint') }}
                    @else
                        {{ __('agentic::agentic.empty.plans_appear') }}
                    @endif
                </core:text>
            </div>
        @endif
    </core:card>
</div>
