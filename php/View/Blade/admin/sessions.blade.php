<div>
    <div class="flex flex-col md:flex-row md:items-center md:justify-between gap-4 mb-6">
        <div>
            <core:heading size="xl">{{ __('agentic::agentic.sessions.title') }}</core:heading>
            <core:subheading>{{ __('agentic::agentic.sessions.subtitle') }}</core:subheading>
        </div>
        @if($this->activeCount > 0)
            <div class="flex items-center gap-2 px-3 py-2 bg-green-50 dark:bg-green-900/20 border border-green-200 dark:border-green-800 rounded-lg">
                <span class="w-2 h-2 bg-green-500 rounded-full animate-pulse"></span>
                <core:text class="text-green-700 dark:text-green-300 font-medium">{{ __('agentic::agentic.sessions.active_sessions', ['count' => $this->activeCount]) }}</core:text>
            </div>
        @endif
    </div>

    {{-- Filters --}}
    <core:card class="p-4 mb-6">
        <div class="flex flex-col md:flex-row gap-4">
            <div class="flex-1">
                <core:input
                    wire:model.live.debounce.300ms="search"
                    :placeholder="__('agentic::agentic.sessions.search_placeholder')"
                    icon="magnifying-glass"
                />
            </div>
            <div class="w-full md:w-40">
                <core:select wire:model.live="status">
                    <option value="">{{ __('agentic::agentic.filters.all_statuses') }}</option>
                    @foreach($this->statusOptions as $value => $label)
                        <option value="{{ $value }}">{{ $label }}</option>
                    @endforeach
                </core:select>
            </div>
            <div class="w-full md:w-40">
                <core:select wire:model.live="agentType">
                    <option value="">{{ __('agentic::agentic.filters.all_agents') }}</option>
                    @foreach($this->agentTypes as $value => $label)
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
            <div class="w-full md:w-48">
                <core:select wire:model.live="planSlug">
                    <option value="">{{ __('agentic::agentic.filters.all_plans') }}</option>
                    @foreach($this->plans as $plan)
                        <option value="{{ $plan->slug }}">{{ $plan->title }}</option>
                    @endforeach
                </core:select>
            </div>
            @if($search || $status || $agentType || $workspace || $planSlug)
                <core:button wire:click="clearFilters" variant="ghost" icon="x-mark">
                    {{ __('agentic::agentic.actions.clear') }}
                </core:button>
            @endif
        </div>
    </core:card>

    {{-- Sessions Table --}}
    <core:card>
        @if($this->sessions->count() > 0)
            <div class="overflow-x-auto">
                <table class="w-full">
                    <thead>
                        <tr class="border-b border-zinc-200 dark:border-zinc-700">
                            <th class="text-left p-4 font-medium text-zinc-600 dark:text-zinc-300">{{ __('agentic::agentic.table.session') }}</th>
                            <th class="text-left p-4 font-medium text-zinc-600 dark:text-zinc-300">{{ __('agentic::agentic.table.agent') }}</th>
                            <th class="text-left p-4 font-medium text-zinc-600 dark:text-zinc-300">{{ __('agentic::agentic.table.plan') }}</th>
                            <th class="text-left p-4 font-medium text-zinc-600 dark:text-zinc-300">{{ __('agentic::agentic.table.status') }}</th>
                            <th class="text-left p-4 font-medium text-zinc-600 dark:text-zinc-300">{{ __('agentic::agentic.table.duration') }}</th>
                            <th class="text-left p-4 font-medium text-zinc-600 dark:text-zinc-300">{{ __('agentic::agentic.table.activity') }}</th>
                            <th class="text-left p-4 font-medium text-zinc-600 dark:text-zinc-300">{{ __('agentic::agentic.table.actions') }}</th>
                            <th class="text-right p-4 font-medium text-zinc-600 dark:text-zinc-300"></th>
                        </tr>
                    </thead>
                    <tbody class="divide-y divide-zinc-200 dark:divide-zinc-700">
                        @foreach($this->sessions as $session)
                            <tr class="hover:bg-zinc-50 dark:hover:bg-zinc-800/50 {{ $session->isActive() ? 'bg-green-50/30 dark:bg-green-900/10' : '' }}">
                                <td class="p-4">
                                    <a href="{{ route('hub.agents.sessions.show', $session->id) }}" wire:navigate class="block">
                                        <code class="text-sm bg-zinc-100 dark:bg-zinc-800 px-2 py-1 rounded hover:bg-violet-100 dark:hover:bg-violet-900/30 transition-colors">{{ $session->session_id }}</code>
                                    </a>
                                    <core:text size="sm" class="text-zinc-500 mt-1">{{ $session->workspace?->name ?? 'N/A' }}</core:text>
                                </td>
                                <td class="p-4">
                                    @if($session->agent_type)
                                        <span class="inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium {{ $this->getAgentBadgeClass($session->agent_type) }}">
                                            {{ ucfirst($session->agent_type) }}
                                        </span>
                                    @else
                                        <core:text size="sm" class="text-zinc-400">{{ __('agentic::agentic.sessions.unknown_agent') }}</core:text>
                                    @endif
                                </td>
                                <td class="p-4">
                                    @if($session->plan)
                                        <a href="{{ route('hub.agents.plans.show', $session->plan->slug) }}" wire:navigate class="text-violet-600 hover:text-violet-500 text-sm">
                                            {{ $session->plan->title }}
                                        </a>
                                    @else
                                        <core:text size="sm" class="text-zinc-400">{{ __('agentic::agentic.sessions.no_plan') }}</core:text>
                                    @endif
                                </td>
                                <td class="p-4">
                                    <div class="flex items-center gap-2">
                                        @if($session->isActive())
                                            <span class="w-2 h-2 bg-green-500 rounded-full animate-pulse"></span>
                                        @endif
                                        <span class="inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium {{ $this->getStatusColorClass($session->status) }}">
                                            {{ ucfirst($session->status) }}
                                        </span>
                                    </div>
                                </td>
                                <td class="p-4">
                                    <core:text size="sm">{{ $session->getDurationFormatted() }}</core:text>
                                </td>
                                <td class="p-4">
                                    <div class="flex items-center gap-2">
                                        <core:text size="sm" class="text-zinc-500">{{ __('agentic::agentic.sessions.actions_count', ['count' => count($session->work_log ?? [])]) }}</core:text>
                                        <core:text size="sm" class="text-zinc-400">&middot;</core:text>
                                        <core:text size="sm" class="text-zinc-500">{{ __('agentic::agentic.sessions.artifacts_count', ['count' => count($session->artifacts ?? [])]) }}</core:text>
                                    </div>
                                    <core:text size="xs" class="text-zinc-400 mt-1">Last: {{ $session->last_active_at?->diffForHumans() ?? 'N/A' }}</core:text>
                                </td>
                                <td class="p-4">
                                    @if($session->isActive())
                                        <core:button wire:click="pause({{ $session->id }})" size="sm" variant="ghost" icon="pause">{{ __('agentic::agentic.actions.pause') }}</core:button>
                                    @elseif($session->isPaused())
                                        <core:button wire:click="resume({{ $session->id }})" size="sm" variant="ghost" icon="play">{{ __('agentic::agentic.actions.resume') }}</core:button>
                                    @endif
                                </td>
                                <td class="p-4 text-right">
                                    <div class="flex items-center justify-end gap-2">
                                        <a href="{{ route('hub.agents.sessions.show', $session->id) }}" wire:navigate>
                                            <core:button variant="ghost" size="sm" icon="eye">{{ __('agentic::agentic.actions.view') }}</core:button>
                                        </a>
                                        @if(!$session->isEnded())
                                            <core:dropdown>
                                                <core:button variant="ghost" size="sm" icon="ellipsis-vertical" />
                                                <core:menu>
                                                    @if($session->isActive())
                                                        <core:menu.item wire:click="pause({{ $session->id }})" icon="pause">{{ __('agentic::agentic.actions.pause') }}</core:menu.item>
                                                    @endif
                                                    @if($session->isPaused())
                                                        <core:menu.item wire:click="resume({{ $session->id }})" icon="play">{{ __('agentic::agentic.actions.resume') }}</core:menu.item>
                                                    @endif
                                                    <core:menu.separator />
                                                    <core:menu.item wire:click="complete({{ $session->id }})" icon="check">{{ __('agentic::agentic.actions.complete') }}</core:menu.item>
                                                    <core:menu.item wire:click="fail({{ $session->id }})" icon="x-mark" variant="danger">{{ __('agentic::agentic.actions.fail') }}</core:menu.item>
                                                </core:menu>
                                            </core:dropdown>
                                        @endif
                                    </div>
                                </td>
                            </tr>
                        @endforeach
                    </tbody>
                </table>
            </div>

            {{-- Pagination --}}
            <div class="p-4 border-t border-zinc-200 dark:border-zinc-700">
                {{ $this->sessions->links() }}
            </div>
        @else
            <div class="flex flex-col items-center py-12 text-center">
                <core:icon name="play" class="w-16 h-16 text-zinc-300 dark:text-zinc-600 mb-4" />
                <core:heading size="lg" class="text-zinc-600 dark:text-zinc-400">{{ __('agentic::agentic.empty.no_sessions') }}</core:heading>
                <core:text class="text-zinc-500 mt-2">
                    @if($search || $status || $agentType || $workspace || $planSlug)
                        {{ __('agentic::agentic.empty.filter_hint') }}
                    @else
                        {{ __('agentic::agentic.empty.sessions_appear') }}
                    @endif
                </core:text>
            </div>
        @endif
    </core:card>
</div>
