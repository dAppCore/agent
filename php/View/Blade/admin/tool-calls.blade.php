<div>
    {{-- Header --}}
    <div class="flex flex-col md:flex-row md:items-center md:justify-between gap-4 mb-6">
        <div>
            <div class="flex items-center gap-3 mb-2">
                <a href="{{ route('hub.agents.tools') }}" wire:navigate class="text-zinc-500 hover:text-zinc-700 dark:hover:text-zinc-300">
                    <core:icon name="arrow-left" class="w-5 h-5" />
                </a>
                <core:heading size="xl">{{ __('agentic::agentic.tool_calls.title') }}</core:heading>
            </div>
            <core:subheading>{{ __('agentic::agentic.tool_calls.subtitle') }}</core:subheading>
        </div>
    </div>

    {{-- Filters --}}
    <core:card class="p-4 mb-6">
        <div class="flex flex-col md:flex-row gap-4">
            <div class="flex-1">
                <core:input
                    wire:model.live.debounce.300ms="search"
                    placeholder="{{ __('agentic::agentic.tool_calls.search_placeholder') }}"
                    icon="magnifying-glass"
                />
            </div>
            <div class="w-full md:w-40">
                <core:select wire:model.live="server">
                    <option value="">{{ __('agentic::agentic.filters.all_servers') }}</option>
                    @foreach($this->servers as $srv)
                        <option value="{{ $srv }}">{{ $srv }}</option>
                    @endforeach
                </core:select>
            </div>
            <div class="w-full md:w-48">
                <core:select wire:model.live="tool">
                    <option value="">{{ __('agentic::agentic.filters.all_tools') }}</option>
                    @foreach($this->tools as $t)
                        <option value="{{ $t }}">{{ $t }}</option>
                    @endforeach
                </core:select>
            </div>
            <div class="w-full md:w-32">
                <core:select wire:model.live="status">
                    <option value="">{{ __('agentic::agentic.filters.all_status') }}</option>
                    <option value="success">{{ __('agentic::agentic.filters.success') }}</option>
                    <option value="failed">{{ __('agentic::agentic.filters.failed') }}</option>
                </core:select>
            </div>
            <div class="w-full md:w-40">
                <core:select wire:model.live="workspace">
                    <option value="">{{ __('agentic::agentic.filters.all_workspaces') }}</option>
                    @foreach($this->workspaces as $ws)
                        <option value="{{ $ws->id }}">{{ $ws->name }}</option>
                    @endforeach
                </core:select>
            </div>
            <div class="w-full md:w-32">
                <core:select wire:model.live="agentType">
                    <option value="">{{ __('agentic::agentic.filters.all_agents') }}</option>
                    @foreach($this->agentTypes as $value => $label)
                        <option value="{{ $value }}">{{ $label }}</option>
                    @endforeach
                </core:select>
            </div>
            @if($search || $server || $tool || $status || $workspace || $agentType)
                <core:button wire:click="clearFilters" variant="ghost" icon="x-mark">
                    {{ __('agentic::agentic.actions.clear') }}
                </core:button>
            @endif
        </div>
    </core:card>

    {{-- Calls Table --}}
    <core:card>
        @if($this->calls->count() > 0)
            <div class="overflow-x-auto">
                <table class="w-full">
                    <thead>
                        <tr class="border-b border-zinc-200 dark:border-zinc-700">
                            <th class="text-left p-4 font-medium text-zinc-600 dark:text-zinc-300">{{ __('agentic::agentic.table.tool') }}</th>
                            <th class="text-left p-4 font-medium text-zinc-600 dark:text-zinc-300">{{ __('agentic::agentic.table.server') }}</th>
                            <th class="text-left p-4 font-medium text-zinc-600 dark:text-zinc-300">{{ __('agentic::agentic.table.status') }}</th>
                            <th class="text-left p-4 font-medium text-zinc-600 dark:text-zinc-300">{{ __('agentic::agentic.table.duration') }}</th>
                            <th class="text-left p-4 font-medium text-zinc-600 dark:text-zinc-300">{{ __('agentic::agentic.table.agent') }}</th>
                            <th class="text-left p-4 font-medium text-zinc-600 dark:text-zinc-300">{{ __('agentic::agentic.table.workspace') }}</th>
                            <th class="text-left p-4 font-medium text-zinc-600 dark:text-zinc-300">{{ __('agentic::agentic.table.time') }}</th>
                            <th class="text-right p-4 font-medium text-zinc-600 dark:text-zinc-300"></th>
                        </tr>
                    </thead>
                    <tbody class="divide-y divide-zinc-200 dark:divide-zinc-700">
                        @foreach($this->calls as $call)
                            <tr class="hover:bg-zinc-50 dark:hover:bg-zinc-800/50 {{ !$call->success ? 'bg-red-50/30 dark:bg-red-900/10' : '' }}">
                                <td class="p-4">
                                    <core:text class="font-medium">{{ $call->tool_name }}</core:text>
                                    @if($call->session_id)
                                        <core:text size="xs" class="text-zinc-400 mt-1">{{ Str::limit($call->session_id, 20) }}</core:text>
                                    @endif
                                </td>
                                <td class="p-4">
                                    <code class="text-xs bg-zinc-100 dark:bg-zinc-800 px-2 py-1 rounded">{{ $call->server_id }}</code>
                                </td>
                                <td class="p-4">
                                    <span class="inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium {{ $this->getStatusBadgeClass($call->success) }}">
                                        {{ $call->success ? __('agentic::agentic.status.success') : __('agentic::agentic.status.failed') }}
                                    </span>
                                </td>
                                <td class="p-4">
                                    <core:text size="sm">{{ $call->getDurationForHumans() }}</core:text>
                                </td>
                                <td class="p-4">
                                    @if($call->agent_type)
                                        <span class="inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium {{ $this->getAgentBadgeClass($call->agent_type) }}">
                                            {{ ucfirst($call->agent_type) }}
                                        </span>
                                    @else
                                        <core:text size="sm" class="text-zinc-400">-</core:text>
                                    @endif
                                </td>
                                <td class="p-4">
                                    <core:text size="sm" class="text-zinc-500">{{ $call->workspace?->name ?? '-' }}</core:text>
                                </td>
                                <td class="p-4">
                                    <core:text size="sm" class="text-zinc-500">{{ $call->created_at->diffForHumans() }}</core:text>
                                    <core:text size="xs" class="text-zinc-400">{{ $call->created_at->format('M j, H:i') }}</core:text>
                                </td>
                                <td class="p-4 text-right">
                                    <core:button wire:click="viewCall({{ $call->id }})" variant="ghost" size="sm" icon="eye">
                                        {{ __('agentic::agentic.tool_calls.details') }}
                                    </core:button>
                                </td>
                            </tr>
                        @endforeach
                    </tbody>
                </table>
            </div>

            {{-- Pagination --}}
            <div class="p-4 border-t border-zinc-200 dark:border-zinc-700">
                {{ $this->calls->links() }}
            </div>
        @else
            <div class="flex flex-col items-center py-12 text-center">
                <core:icon name="wrench" class="w-16 h-16 text-zinc-300 dark:text-zinc-600 mb-4" />
                <core:heading size="lg" class="text-zinc-600 dark:text-zinc-400">{{ __('agentic::agentic.tool_calls.no_calls') }}</core:heading>
                <core:text class="text-zinc-500 mt-2">
                    @if($search || $server || $tool || $status || $workspace || $agentType)
                        {{ __('agentic::agentic.tool_calls.no_calls_filtered') }}
                    @else
                        {{ __('agentic::agentic.tool_calls.no_calls_empty') }}
                    @endif
                </core:text>
            </div>
        @endif
    </core:card>

    {{-- Call Detail Modal (AC18) --}}
    @if($this->selectedCall)
        <core:modal wire:model.self="selectedCallId" class="max-w-4xl">
            <div class="p-6">
                <div class="flex items-start justify-between mb-6">
                    <div>
                        <core:heading size="lg">{{ $this->selectedCall->tool_name }}</core:heading>
                        <div class="flex items-center gap-2 mt-2">
                            <code class="text-xs bg-zinc-100 dark:bg-zinc-800 px-2 py-1 rounded">{{ $this->selectedCall->server_id }}</code>
                            <span class="inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium {{ $this->getStatusBadgeClass($this->selectedCall->success) }}">
                                {{ $this->selectedCall->success ? __('agentic::agentic.status.success') : __('agentic::agentic.status.failed') }}
                            </span>
                        </div>
                    </div>
                    <core:button wire:click="closeCallDetail" variant="ghost" icon="x-mark" />
                </div>

                {{-- Metadata --}}
                <div class="grid grid-cols-2 md:grid-cols-4 gap-4 mb-6">
                    <div>
                        <core:text size="sm" class="text-zinc-500">{{ __('agentic::agentic.tool_calls.metadata.duration') }}</core:text>
                        <core:text class="font-medium">{{ $this->selectedCall->getDurationForHumans() }}</core:text>
                    </div>
                    <div>
                        <core:text size="sm" class="text-zinc-500">{{ __('agentic::agentic.tool_calls.metadata.agent_type') }}</core:text>
                        <core:text class="font-medium">{{ ucfirst($this->selectedCall->agent_type ?? 'Unknown') }}</core:text>
                    </div>
                    <div>
                        <core:text size="sm" class="text-zinc-500">{{ __('agentic::agentic.tool_calls.metadata.workspace') }}</core:text>
                        <core:text class="font-medium">{{ $this->selectedCall->workspace?->name ?? 'N/A' }}</core:text>
                    </div>
                    <div>
                        <core:text size="sm" class="text-zinc-500">{{ __('agentic::agentic.tool_calls.metadata.time') }}</core:text>
                        <core:text class="font-medium">{{ $this->selectedCall->created_at->format('M j, Y H:i:s') }}</core:text>
                    </div>
                </div>

                @if($this->selectedCall->session_id)
                    <div class="mb-6">
                        <core:text size="sm" class="text-zinc-500 mb-1">{{ __('agentic::agentic.tool_calls.session_id') }}</core:text>
                        <code class="text-sm bg-zinc-100 dark:bg-zinc-800 px-2 py-1 rounded block overflow-x-auto">{{ $this->selectedCall->session_id }}</code>
                    </div>
                @endif

                @if($this->selectedCall->plan_slug)
                    <div class="mb-6">
                        <core:text size="sm" class="text-zinc-500 mb-1">{{ __('agentic::agentic.table.plan') }}</core:text>
                        <a href="{{ route('hub.agents.plans.show', $this->selectedCall->plan_slug) }}" wire:navigate class="text-violet-600 hover:text-violet-500">
                            {{ $this->selectedCall->plan_slug }}
                        </a>
                    </div>
                @endif

                {{-- Input Parameters --}}
                @if($this->selectedCall->input_params && count($this->selectedCall->input_params) > 0)
                    <div class="mb-6">
                        <core:text size="sm" class="text-zinc-500 font-medium mb-2">{{ __('agentic::agentic.tool_calls.input_params') }}</core:text>
                        <div class="bg-zinc-100 dark:bg-zinc-800 rounded-lg p-4 overflow-x-auto">
                            <pre class="text-sm text-zinc-800 dark:text-zinc-200 whitespace-pre-wrap">{{ json_encode($this->selectedCall->input_params, JSON_PRETTY_PRINT | JSON_UNESCAPED_SLASHES) }}</pre>
                        </div>
                    </div>
                @endif

                {{-- Error Details --}}
                @if(!$this->selectedCall->success)
                    <div class="mb-6 p-4 bg-red-50 dark:bg-red-900/20 border border-red-200 dark:border-red-800 rounded-lg">
                        <core:text size="sm" class="text-red-700 dark:text-red-300 font-medium mb-2">{{ __('agentic::agentic.tool_calls.error_details') }}</core:text>
                        @if($this->selectedCall->error_code)
                            <core:text size="sm" class="text-red-600 dark:text-red-400 mb-1">{{ __('agentic::agentic.tools.error_code', ['code' => $this->selectedCall->error_code]) }}</core:text>
                        @endif
                        <core:text class="text-red-700 dark:text-red-300">{{ $this->selectedCall->error_message ?? __('agentic::agentic.tools.unknown_error') }}</core:text>
                    </div>
                @endif

                {{-- Result Summary --}}
                @if($this->selectedCall->result_summary && count($this->selectedCall->result_summary) > 0)
                    <div class="mb-6">
                        <core:text size="sm" class="text-zinc-500 font-medium mb-2">{{ __('agentic::agentic.tool_calls.result_summary') }}</core:text>
                        <div class="bg-zinc-100 dark:bg-zinc-800 rounded-lg p-4 overflow-x-auto">
                            <pre class="text-sm text-zinc-800 dark:text-zinc-200 whitespace-pre-wrap">{{ json_encode($this->selectedCall->result_summary, JSON_PRETTY_PRINT | JSON_UNESCAPED_SLASHES) }}</pre>
                        </div>
                    </div>
                @endif

                <div class="flex justify-end gap-2">
                    <core:button wire:click="closeCallDetail" variant="ghost">{{ __('agentic::agentic.actions.close') }}</core:button>
                </div>
            </div>
        </core:modal>
    @endif
</div>
