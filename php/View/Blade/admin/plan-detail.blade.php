<div>
    {{-- Header --}}
    <div class="flex flex-col md:flex-row md:items-center md:justify-between gap-4 mb-6">
        <div>
            <div class="flex items-center gap-2 mb-2">
                <a href="{{ route('hub.agents.plans') }}" wire:navigate class="text-zinc-500 hover:text-zinc-700 dark:hover:text-zinc-300">
                    <core:icon name="arrow-left" class="w-5 h-5" />
                </a>
                <core:heading size="xl">{{ $plan->title }}</core:heading>
                <span class="inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium {{ $this->getStatusColorClass($plan->status) }}">
                    {{ ucfirst($plan->status) }}
                </span>
            </div>
            <core:subheading>{{ $plan->workspace?->name ?? 'No workspace' }} &middot; {{ $plan->slug }}</core:subheading>
        </div>
        <div class="flex items-center gap-2">
            @if($plan->status === 'draft')
                <core:button wire:click="activatePlan" variant="primary" icon="play">{{ __('agentic::agentic.actions.activate') }}</core:button>
            @endif
            @if($plan->status === 'active')
                <core:button wire:click="completePlan" variant="primary" icon="check">{{ __('agentic::agentic.actions.complete') }}</core:button>
            @endif
            @if($plan->status !== 'archived')
                <core:button wire:click="archivePlan" wire:confirm="{{ __('agentic::agentic.confirm.archive_plan') }}" variant="ghost" icon="archive-box">{{ __('agentic::agentic.actions.archive') }}</core:button>
            @endif
        </div>
    </div>

    {{-- Progress Overview --}}
    <core:card class="p-6 mb-6">
        <div class="flex items-center justify-between mb-4">
            <core:heading size="lg">{{ __('agentic::agentic.plan_detail.progress') }}</core:heading>
            <core:text class="text-lg font-semibold text-violet-600 dark:text-violet-400">{{ $this->progress['percentage'] }}%</core:text>
        </div>
        <div class="w-full bg-zinc-200 dark:bg-zinc-700 rounded-full h-4 mb-4">
            <div class="bg-violet-500 h-4 rounded-full transition-all" style="width: {{ $this->progress['percentage'] }}%"></div>
        </div>
        <div class="grid grid-cols-4 gap-4 text-center">
            <div>
                <core:text size="lg" class="font-semibold">{{ $this->progress['total'] }}</core:text>
                <core:text size="sm" class="text-zinc-500">{{ __('agentic::agentic.plans.total_phases') }}</core:text>
            </div>
            <div>
                <core:text size="lg" class="font-semibold text-green-600 dark:text-green-400">{{ $this->progress['completed'] }}</core:text>
                <core:text size="sm" class="text-zinc-500">{{ __('agentic::agentic.plans.completed') }}</core:text>
            </div>
            <div>
                <core:text size="lg" class="font-semibold text-blue-600 dark:text-blue-400">{{ $this->progress['in_progress'] }}</core:text>
                <core:text size="sm" class="text-zinc-500">{{ __('agentic::agentic.plans.in_progress') }}</core:text>
            </div>
            <div>
                <core:text size="lg" class="font-semibold text-zinc-600 dark:text-zinc-400">{{ $this->progress['pending'] }}</core:text>
                <core:text size="sm" class="text-zinc-500">{{ __('agentic::agentic.plans.pending') }}</core:text>
            </div>
        </div>
    </core:card>

    {{-- Description --}}
    @if($plan->description)
        <core:card class="p-6 mb-6">
            <core:heading size="lg" class="mb-3">{{ __('agentic::agentic.plan_detail.description') }}</core:heading>
            <core:text class="text-zinc-600 dark:text-zinc-300 whitespace-pre-wrap">{{ $plan->description }}</core:text>
        </core:card>
    @endif

    {{-- Phases --}}
    <core:card class="p-6 mb-6">
        <core:heading size="lg" class="mb-4">{{ __('agentic::agentic.plan_detail.phases') }}</core:heading>

        @if($this->phases->count() > 0)
            <div class="space-y-4">
                @foreach($this->phases as $phase)
                    @php
                        $taskProgress = $phase->getTaskProgress();
                        $statusIcon = $phase->getStatusIcon();
                    @endphp
                    <div class="border border-zinc-200 dark:border-zinc-700 rounded-lg overflow-hidden">
                        {{-- Phase Header --}}
                        <div class="flex items-center justify-between p-4 bg-zinc-50 dark:bg-zinc-800/50">
                            <div class="flex items-center gap-3">
                                <span class="text-xl">{{ $statusIcon }}</span>
                                <div>
                                    <div class="flex items-center gap-2">
                                        <core:text class="font-medium">{{ __('agentic::agentic.plan_detail.phase_number', ['number' => $phase->order]) }}: {{ $phase->name }}</core:text>
                                        <span class="inline-flex items-center px-2 py-0.5 rounded-full text-xs font-medium {{ $this->getStatusColorClass($phase->status) }}">
                                            {{ ucfirst(str_replace('_', ' ', $phase->status)) }}
                                        </span>
                                    </div>
                                    @if($phase->description)
                                        <core:text size="sm" class="text-zinc-500 mt-1">{{ $phase->description }}</core:text>
                                    @endif
                                </div>
                            </div>
                            <div class="flex items-center gap-2">
                                {{-- Phase Progress --}}
                                @if($taskProgress['total'] > 0)
                                    <div class="flex items-center gap-2 mr-4">
                                        <div class="w-20 bg-zinc-200 dark:bg-zinc-700 rounded-full h-2">
                                            <div class="bg-violet-500 h-2 rounded-full" style="width: {{ $taskProgress['percentage'] }}%"></div>
                                        </div>
                                        <core:text size="sm" class="text-zinc-500">{{ __('agentic::agentic.plan_detail.tasks_progress', ['completed' => $taskProgress['completed'], 'total' => $taskProgress['total']]) }}</core:text>
                                    </div>
                                @endif

                                {{-- Phase Actions --}}
                                <core:dropdown>
                                    <core:button variant="ghost" size="sm" icon="ellipsis-vertical" />
                                    <core:menu>
                                        @if($phase->isPending())
                                            <core:menu.item wire:click="startPhase({{ $phase->id }})" icon="play">{{ __('agentic::agentic.actions.start_phase') }}</core:menu.item>
                                        @endif
                                        @if($phase->isInProgress())
                                            <core:menu.item wire:click="completePhase({{ $phase->id }})" icon="check">{{ __('agentic::agentic.actions.complete_phase') }}</core:menu.item>
                                            <core:menu.item wire:click="blockPhase({{ $phase->id }})" icon="exclamation-triangle">{{ __('agentic::agentic.actions.block_phase') }}</core:menu.item>
                                        @endif
                                        @if($phase->isBlocked())
                                            <core:menu.item wire:click="resetPhase({{ $phase->id }})" icon="arrow-path">{{ __('agentic::agentic.actions.unblock') }}</core:menu.item>
                                        @endif
                                        @if(!$phase->isCompleted() && !$phase->isSkipped())
                                            <core:menu.item wire:click="skipPhase({{ $phase->id }})" icon="forward">{{ __('agentic::agentic.actions.skip_phase') }}</core:menu.item>
                                        @endif
                                        @if($phase->isCompleted() || $phase->isSkipped())
                                            <core:menu.item wire:click="resetPhase({{ $phase->id }})" icon="arrow-path">{{ __('agentic::agentic.actions.reset_to_pending') }}</core:menu.item>
                                        @endif
                                        <core:menu.separator />
                                        <core:menu.item wire:click="openAddTaskModal({{ $phase->id }})" icon="plus">{{ __('agentic::agentic.actions.add_task') }}</core:menu.item>
                                    </core:menu>
                                </core:dropdown>
                            </div>
                        </div>

                        {{-- Tasks --}}
                        @if($phase->tasks && count($phase->tasks) > 0)
                            <div class="p-4 space-y-2">
                                @foreach($phase->tasks as $index => $task)
                                    @php
                                        $taskName = is_string($task) ? $task : ($task['name'] ?? 'Unknown task');
                                        $taskStatus = is_string($task) ? 'pending' : ($task['status'] ?? 'pending');
                                        $taskNotes = is_array($task) ? ($task['notes'] ?? null) : null;
                                        $isCompleted = $taskStatus === 'completed';
                                    @endphp
                                    <div class="flex items-start gap-3 p-2 rounded hover:bg-zinc-50 dark:hover:bg-zinc-800/50">
                                        <button
                                            wire:click="completeTask({{ $phase->id }}, {{ is_int($index) ? $index : "'{$index}'" }})"
                                            class="flex-shrink-0 mt-0.5"
                                            @if($isCompleted) disabled @endif
                                        >
                                            @if($isCompleted)
                                                <span class="w-5 h-5 flex items-center justify-center rounded-full bg-green-500 text-white">
                                                    <core:icon name="check" class="w-3 h-3" />
                                                </span>
                                            @else
                                                <span class="w-5 h-5 flex items-center justify-center rounded-full border-2 border-zinc-300 dark:border-zinc-600 hover:border-violet-500 transition-colors">
                                                </span>
                                            @endif
                                        </button>
                                        <div class="flex-1">
                                            <core:text class="{{ $isCompleted ? 'line-through text-zinc-400' : '' }}">{{ $taskName }}</core:text>
                                            @if($taskNotes)
                                                <core:text size="sm" class="text-zinc-500 mt-1">{{ $taskNotes }}</core:text>
                                            @endif
                                        </div>
                                    </div>
                                @endforeach
                            </div>
                        @else
                            <div class="p-4 text-center">
                                <core:text size="sm" class="text-zinc-500">{{ __('agentic::agentic.plans.no_tasks') }}</core:text>
                                <button wire:click="openAddTaskModal({{ $phase->id }})" class="text-violet-600 hover:text-violet-500 text-sm mt-1">
                                    {{ __('agentic::agentic.plans.add_task') }}
                                </button>
                            </div>
                        @endif
                    </div>
                @endforeach
            </div>
        @else
            <div class="text-center py-8">
                <core:icon name="clipboard-document-list" class="w-12 h-12 text-zinc-300 dark:text-zinc-600 mx-auto mb-3" />
                <core:text class="text-zinc-500">{{ __('agentic::agentic.plan_detail.no_phases') }}</core:text>
            </div>
        @endif
    </core:card>

    {{-- Sessions --}}
    <core:card class="p-6">
        <div class="flex items-center justify-between mb-4">
            <core:heading size="lg">{{ __('agentic::agentic.plan_detail.sessions') }}</core:heading>
            <core:text size="sm" class="text-zinc-500">{{ $this->sessions->count() }} session(s)</core:text>
        </div>

        @if($this->sessions->count() > 0)
            <div class="overflow-x-auto">
                <table class="w-full">
                    <thead>
                        <tr class="border-b border-zinc-200 dark:border-zinc-700">
                            <th class="text-left p-3 font-medium text-zinc-600 dark:text-zinc-300">{{ __('agentic::agentic.table.session') }}</th>
                            <th class="text-left p-3 font-medium text-zinc-600 dark:text-zinc-300">{{ __('agentic::agentic.table.agent') }}</th>
                            <th class="text-left p-3 font-medium text-zinc-600 dark:text-zinc-300">{{ __('agentic::agentic.table.status') }}</th>
                            <th class="text-left p-3 font-medium text-zinc-600 dark:text-zinc-300">{{ __('agentic::agentic.table.duration') }}</th>
                            <th class="text-left p-3 font-medium text-zinc-600 dark:text-zinc-300">{{ __('agentic::agentic.session_detail.started') }}</th>
                            <th class="text-right p-3 font-medium text-zinc-600 dark:text-zinc-300">{{ __('agentic::agentic.table.actions') }}</th>
                        </tr>
                    </thead>
                    <tbody class="divide-y divide-zinc-200 dark:divide-zinc-700">
                        @foreach($this->sessions as $session)
                            <tr class="hover:bg-zinc-50 dark:hover:bg-zinc-800/50">
                                <td class="p-3">
                                    <code class="text-sm bg-zinc-100 dark:bg-zinc-800 px-2 py-1 rounded">{{ $session->session_id }}</code>
                                </td>
                                <td class="p-3">
                                    <core:text size="sm">{{ $session->agent_type ?? __('agentic::agentic.sessions.unknown_agent') }}</core:text>
                                </td>
                                <td class="p-3">
                                    <span class="inline-flex items-center px-2 py-0.5 rounded-full text-xs font-medium
                                        @if($session->status === 'active') bg-green-100 text-green-700 dark:bg-green-900/50 dark:text-green-300
                                        @elseif($session->status === 'paused') bg-amber-100 text-amber-700 dark:bg-amber-900/50 dark:text-amber-300
                                        @elseif($session->status === 'completed') bg-blue-100 text-blue-700 dark:bg-blue-900/50 dark:text-blue-300
                                        @else bg-red-100 text-red-700 dark:bg-red-900/50 dark:text-red-300
                                        @endif">
                                        {{ ucfirst($session->status) }}
                                    </span>
                                </td>
                                <td class="p-3">
                                    <core:text size="sm">{{ $session->getDurationFormatted() }}</core:text>
                                </td>
                                <td class="p-3">
                                    <core:text size="sm" class="text-zinc-500">{{ $session->started_at?->diffForHumans() ?? 'N/A' }}</core:text>
                                </td>
                                <td class="p-3 text-right">
                                    <a href="{{ route('hub.agents.sessions.show', $session->id) }}" wire:navigate>
                                        <core:button variant="ghost" size="sm" icon="eye">{{ __('agentic::agentic.actions.view') }}</core:button>
                                    </a>
                                </td>
                            </tr>
                        @endforeach
                    </tbody>
                </table>
            </div>
        @else
            <div class="text-center py-8">
                <core:icon name="play" class="w-12 h-12 text-zinc-300 dark:text-zinc-600 mx-auto mb-3" />
                <core:text class="text-zinc-500">{{ __('agentic::agentic.plan_detail.no_sessions') }}</core:text>
            </div>
        @endif
    </core:card>

    {{-- Add Task Modal --}}
    <core:modal wire:model="showAddTaskModal" class="max-w-md">
        <div class="p-6">
            <core:heading size="lg" class="mb-4">{{ __('agentic::agentic.add_task.title') }}</core:heading>

            <form wire:submit="addTask" class="space-y-4">
                <core:input
                    wire:model="newTaskName"
                    label="{{ __('agentic::agentic.add_task.task_name') }}"
                    placeholder="{{ __('agentic::agentic.add_task.task_name_placeholder') }}"
                    required
                />

                <core:textarea
                    wire:model="newTaskNotes"
                    label="{{ __('agentic::agentic.add_task.notes') }}"
                    placeholder="{{ __('agentic::agentic.add_task.notes_placeholder') }}"
                    rows="3"
                />

                <div class="flex justify-end gap-2 pt-4">
                    <core:button type="button" wire:click="$set('showAddTaskModal', false)" variant="ghost">{{ __('agentic::agentic.actions.cancel') }}</core:button>
                    <core:button type="submit" variant="primary">{{ __('agentic::agentic.actions.add_task') }}</core:button>
                </div>
            </form>
        </div>
    </core:modal>
</div>
