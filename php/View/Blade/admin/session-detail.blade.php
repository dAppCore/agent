<div wire:poll.{{ $pollingInterval }}ms="poll">
    {{-- Header --}}
    <div class="flex flex-col md:flex-row md:items-start md:justify-between gap-4 mb-6">
        <div>
            <div class="flex items-center gap-3 mb-2">
                <a href="{{ route('hub.agents.sessions') }}" wire:navigate class="text-zinc-500 hover:text-zinc-700 dark:hover:text-zinc-300">
                    <core:icon name="arrow-left" class="w-5 h-5" />
                </a>
                <core:heading size="xl">{{ __('agentic::agentic.session_detail.title') }}</core:heading>
            </div>
            <div class="flex items-center gap-3">
                <code class="text-lg bg-zinc-100 dark:bg-zinc-800 px-3 py-1 rounded">{{ $session->session_id }}</code>
                @if($session->isActive())
                    <span class="w-2 h-2 bg-green-500 rounded-full animate-pulse"></span>
                @endif
                <span class="inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium {{ $this->getStatusColorClass($session->status) }}">
                    {{ ucfirst($session->status) }}
                </span>
                @if($session->agent_type)
                    <span class="inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium {{ $this->getAgentBadgeClass($session->agent_type) }}">
                        {{ ucfirst($session->agent_type) }}
                    </span>
                @endif
            </div>
        </div>

        {{-- Actions --}}
        <div class="flex items-center gap-2">
            @if($session->isActive())
                <core:button wire:click="pauseSession" variant="ghost" icon="pause">{{ __('agentic::agentic.actions.pause') }}</core:button>
            @elseif($session->isPaused())
                <core:button wire:click="resumeSession" variant="ghost" icon="play">{{ __('agentic::agentic.actions.resume') }}</core:button>
            @endif

            {{-- Replay button - available for any session with work log --}}
            @if(count($this->workLog) > 0)
                <core:button wire:click="openReplayModal" variant="ghost" icon="arrow-path">{{ __('agentic::agentic.actions.replay') }}</core:button>
            @endif

            @if(!$session->isEnded())
                <core:button wire:click="openCompleteModal" variant="primary" icon="check">{{ __('agentic::agentic.actions.complete') }}</core:button>
                <core:button wire:click="openFailModal" variant="danger" icon="x-mark">{{ __('agentic::agentic.actions.fail') }}</core:button>
            @endif
        </div>
    </div>

    {{-- Session Info Cards --}}
    <div class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-4 mb-6">
        <core:card class="p-4">
            <core:text size="sm" class="text-zinc-500 mb-1">{{ __('agentic::agentic.session_detail.workspace') }}</core:text>
            <core:text class="font-medium">{{ $session->workspace?->name ?? 'N/A' }}</core:text>
        </core:card>

        <core:card class="p-4">
            <core:text size="sm" class="text-zinc-500 mb-1">{{ __('agentic::agentic.session_detail.plan') }}</core:text>
            @if($session->plan)
                <a href="{{ route('hub.agents.plans.show', $session->plan->slug) }}" wire:navigate class="text-violet-600 hover:text-violet-500 font-medium">
                    {{ $session->plan->title }}
                </a>
            @else
                <core:text class="font-medium text-zinc-400">{{ __('agentic::agentic.sessions.no_plan') }}</core:text>
            @endif
        </core:card>

        <core:card class="p-4">
            <core:text size="sm" class="text-zinc-500 mb-1">{{ __('agentic::agentic.session_detail.duration') }}</core:text>
            <core:text class="font-medium">{{ $session->getDurationFormatted() }}</core:text>
        </core:card>

        <core:card class="p-4">
            <core:text size="sm" class="text-zinc-500 mb-1">{{ __('agentic::agentic.session_detail.activity') }}</core:text>
            <core:text class="font-medium">{{ __('agentic::agentic.sessions.actions_count', ['count' => count($this->workLog)]) }} &middot; {{ __('agentic::agentic.sessions.artifacts_count', ['count' => count($this->artifacts)]) }}</core:text>
        </core:card>
    </div>

    {{-- Plan Timeline (AC11) --}}
    @if($session->agent_plan_id && $this->planSessions->count() > 1)
        <core:card class="p-4 mb-6">
            <core:heading size="sm" class="mb-4">{{ __('agentic::agentic.session_detail.plan_timeline', ['current' => $this->sessionIndex, 'total' => $this->planSessions->count()]) }}</core:heading>
            <div class="flex items-center gap-2 overflow-x-auto pb-2">
                @foreach($this->planSessions as $index => $planSession)
                    <a
                        href="{{ route('hub.agents.sessions.show', $planSession->id) }}"
                        wire:navigate
                        class="flex items-center gap-2 px-3 py-2 rounded-lg border transition-colors min-w-fit
                            {{ $planSession->id === $session->id
                                ? 'bg-violet-50 dark:bg-violet-900/30 border-violet-300 dark:border-violet-700'
                                : 'bg-zinc-50 dark:bg-zinc-800 border-zinc-200 dark:border-zinc-700 hover:bg-zinc-100 dark:hover:bg-zinc-700' }}"
                    >
                        <span class="w-6 h-6 rounded-full flex items-center justify-center text-xs font-medium
                            {{ $planSession->isEnded()
                                ? ($planSession->status === 'completed' ? 'bg-green-100 text-green-700 dark:bg-green-900/50 dark:text-green-300' : 'bg-red-100 text-red-700 dark:bg-red-900/50 dark:text-red-300')
                                : ($planSession->isActive() ? 'bg-green-100 text-green-700 dark:bg-green-900/50 dark:text-green-300' : 'bg-amber-100 text-amber-700 dark:bg-amber-900/50 dark:text-amber-300') }}">
                            {{ $index + 1 }}
                        </span>
                        <div class="text-sm">
                            <div class="font-medium">{{ ucfirst($planSession->agent_type ?? __('agentic::agentic.sessions.unknown_agent')) }}</div>
                            <div class="text-zinc-500 text-xs">{{ $planSession->started_at?->format('M j, H:i') ?? __('agentic::agentic.session_detail.not_started') }}</div>
                        </div>
                        @if($planSession->isActive())
                            <span class="w-2 h-2 bg-green-500 rounded-full animate-pulse"></span>
                        @endif
                    </a>
                    @if(!$loop->last)
                        <core:icon name="chevron-right" class="w-4 h-4 text-zinc-400 flex-shrink-0" />
                    @endif
                @endforeach
            </div>
        </core:card>
    @endif

    <div class="grid grid-cols-1 lg:grid-cols-3 gap-6">
        {{-- Work Log (Left Column - 2/3) --}}
        <div class="lg:col-span-2 space-y-6">
            {{-- Context Summary (AC10) --}}
            @if($this->contextSummary)
                <core:card>
                    <div class="p-4 border-b border-zinc-200 dark:border-zinc-700">
                        <core:heading size="sm">{{ __('agentic::agentic.session_detail.context_summary') }}</core:heading>
                    </div>
                    <div class="p-4 space-y-3">
                        @if(isset($this->contextSummary['goal']))
                            <div>
                                <core:text size="sm" class="text-zinc-500 font-medium">{{ __('agentic::agentic.session_detail.goal') }}</core:text>
                                <core:text>{{ $this->contextSummary['goal'] }}</core:text>
                            </div>
                        @endif
                        @if(isset($this->contextSummary['progress']))
                            <div>
                                <core:text size="sm" class="text-zinc-500 font-medium">{{ __('agentic::agentic.session_detail.progress') }}</core:text>
                                <core:text>{{ $this->contextSummary['progress'] }}</core:text>
                            </div>
                        @endif
                        @if(isset($this->contextSummary['next_steps']) && is_array($this->contextSummary['next_steps']))
                            <div>
                                <core:text size="sm" class="text-zinc-500 font-medium">{{ __('agentic::agentic.session_detail.next_steps') }}</core:text>
                                <ul class="list-disc list-inside text-zinc-700 dark:text-zinc-300">
                                    @foreach($this->contextSummary['next_steps'] as $step)
                                        <li>{{ $step }}</li>
                                    @endforeach
                                </ul>
                            </div>
                        @endif
                    </div>
                </core:card>
            @endif

            {{-- Work Log Timeline (AC9) --}}
            <core:card>
                <div class="p-4 border-b border-zinc-200 dark:border-zinc-700 flex items-center justify-between">
                    <core:heading size="sm">{{ __('agentic::agentic.session_detail.work_log') }}</core:heading>
                    <core:text size="sm" class="text-zinc-500">{{ __('agentic::agentic.session_detail.entries', ['count' => count($this->workLog)]) }}</core:text>
                </div>
                @if(count($this->recentWorkLog) > 0)
                    <div class="divide-y divide-zinc-200 dark:divide-zinc-700 max-h-[600px] overflow-y-auto">
                        @foreach($this->recentWorkLog as $entry)
                            <div class="p-4 hover:bg-zinc-50 dark:hover:bg-zinc-800/50">
                                <div class="flex items-start gap-3">
                                    <div class="flex-shrink-0 mt-0.5">
                                        <core:icon
                                            name="{{ $this->getLogTypeIcon($entry['type'] ?? 'info') }}"
                                            class="w-5 h-5 {{ $this->getLogTypeColor($entry['type'] ?? 'info') }}"
                                        />
                                    </div>
                                    <div class="flex-1 min-w-0">
                                        <div class="flex items-center gap-2 mb-1">
                                            <core:text class="font-medium">{{ $entry['action'] ?? 'Action' }}</core:text>
                                            @if(isset($entry['type']))
                                                <span class="inline-flex items-center px-2 py-0.5 rounded text-xs font-medium bg-zinc-100 dark:bg-zinc-800 text-zinc-600 dark:text-zinc-400">
                                                    {{ $entry['type'] }}
                                                </span>
                                            @endif
                                        </div>
                                        @if(isset($entry['details']))
                                            <core:text size="sm" class="text-zinc-600 dark:text-zinc-400">{{ $entry['details'] }}</core:text>
                                        @endif
                                        @if(isset($entry['timestamp']))
                                            <core:text size="xs" class="text-zinc-400 mt-1">
                                                {{ \Carbon\Carbon::parse($entry['timestamp'])->format('M j, Y H:i:s') }}
                                            </core:text>
                                        @endif
                                    </div>
                                </div>
                            </div>
                        @endforeach
                    </div>
                @else
                    <div class="p-8 text-center">
                        <core:icon name="document-text" class="w-12 h-12 text-zinc-300 dark:text-zinc-600 mx-auto mb-3" />
                        <core:text class="text-zinc-500">{{ __('agentic::agentic.session_detail.no_work_log') }}</core:text>
                    </div>
                @endif
            </core:card>

            {{-- Final Summary (AC10) --}}
            @if($session->final_summary)
                <core:card>
                    <div class="p-4 border-b border-zinc-200 dark:border-zinc-700">
                        <core:heading size="sm">{{ __('agentic::agentic.session_detail.final_summary') }}</core:heading>
                    </div>
                    <div class="p-4">
                        <core:text class="whitespace-pre-wrap">{{ $session->final_summary }}</core:text>
                    </div>
                </core:card>
            @endif
        </div>

        {{-- Right Column (1/3) --}}
        <div class="space-y-6">
            {{-- Session Timestamps --}}
            <core:card>
                <div class="p-4 border-b border-zinc-200 dark:border-zinc-700">
                    <core:heading size="sm">{{ __('agentic::agentic.session_detail.timestamps') }}</core:heading>
                </div>
                <div class="p-4 space-y-3">
                    <div class="flex justify-between">
                        <core:text size="sm" class="text-zinc-500">{{ __('agentic::agentic.session_detail.started') }}</core:text>
                        <core:text size="sm">{{ $session->started_at?->format('M j, Y H:i') ?? __('agentic::agentic.session_detail.not_started') }}</core:text>
                    </div>
                    <div class="flex justify-between">
                        <core:text size="sm" class="text-zinc-500">{{ __('agentic::agentic.session_detail.last_active') }}</core:text>
                        <core:text size="sm">{{ $session->last_active_at?->diffForHumans() ?? 'N/A' }}</core:text>
                    </div>
                    @if($session->ended_at)
                        <div class="flex justify-between">
                            <core:text size="sm" class="text-zinc-500">{{ __('agentic::agentic.session_detail.ended') }}</core:text>
                            <core:text size="sm">{{ $session->ended_at->format('M j, Y H:i') }}</core:text>
                        </div>
                    @endif
                </div>
            </core:card>

            {{-- Artifacts (AC9) --}}
            <core:card>
                <div class="p-4 border-b border-zinc-200 dark:border-zinc-700">
                    <core:heading size="sm">{{ __('agentic::agentic.session_detail.artifacts') }}</core:heading>
                </div>
                @if(count($this->artifacts) > 0)
                    <div class="divide-y divide-zinc-200 dark:divide-zinc-700">
                        @foreach($this->artifacts as $artifact)
                            <div class="p-4">
                                <div class="flex items-center gap-2 mb-1">
                                    <core:icon name="document" class="w-4 h-4 text-zinc-500" />
                                    <core:text class="font-medium">{{ $artifact['name'] ?? 'Artifact' }}</core:text>
                                </div>
                                @if(isset($artifact['type']))
                                    <span class="inline-flex items-center px-2 py-0.5 rounded text-xs font-medium bg-violet-100 dark:bg-violet-900/50 text-violet-700 dark:text-violet-300">
                                        {{ $artifact['type'] }}
                                    </span>
                                @endif
                                @if(isset($artifact['path']))
                                    <core:text size="sm" class="text-zinc-500 mt-1 truncate">{{ $artifact['path'] }}</core:text>
                                @endif
                            </div>
                        @endforeach
                    </div>
                @else
                    <div class="p-4 text-center">
                        <core:text size="sm" class="text-zinc-500">{{ __('agentic::agentic.session_detail.no_artifacts') }}</core:text>
                    </div>
                @endif
            </core:card>

            {{-- Handoff Notes (AC9) --}}
            <core:card>
                <div class="p-4 border-b border-zinc-200 dark:border-zinc-700">
                    <core:heading size="sm">{{ __('agentic::agentic.session_detail.handoff_notes') }}</core:heading>
                </div>
                @if($this->handoffNotes)
                    <div class="p-4 space-y-3">
                        @if(isset($this->handoffNotes['summary']))
                            <div>
                                <core:text size="sm" class="text-zinc-500 font-medium">{{ __('agentic::agentic.session_detail.summary') }}</core:text>
                                <core:text size="sm">{{ $this->handoffNotes['summary'] }}</core:text>
                            </div>
                        @endif
                        @if(isset($this->handoffNotes['blockers']) && is_array($this->handoffNotes['blockers']) && count($this->handoffNotes['blockers']) > 0)
                            <div>
                                <core:text size="sm" class="text-zinc-500 font-medium">{{ __('agentic::agentic.session_detail.blockers') }}</core:text>
                                <ul class="list-disc list-inside text-sm text-zinc-700 dark:text-zinc-300">
                                    @foreach($this->handoffNotes['blockers'] as $blocker)
                                        <li>{{ $blocker }}</li>
                                    @endforeach
                                </ul>
                            </div>
                        @endif
                        @if(isset($this->handoffNotes['next_agent']))
                            <div>
                                <core:text size="sm" class="text-zinc-500 font-medium">{{ __('agentic::agentic.session_detail.suggested_next_agent') }}</core:text>
                                <span class="inline-flex items-center px-2 py-0.5 rounded text-xs font-medium {{ $this->getAgentBadgeClass($this->handoffNotes['next_agent']) }}">
                                    {{ ucfirst($this->handoffNotes['next_agent']) }}
                                </span>
                            </div>
                        @endif
                    </div>
                @else
                    <div class="p-4 text-center">
                        <core:text size="sm" class="text-zinc-500">{{ __('agentic::agentic.session_detail.no_handoff_notes') }}</core:text>
                    </div>
                @endif
            </core:card>
        </div>
    </div>

    {{-- Complete Modal --}}
    <core:modal wire:model="showCompleteModal" class="max-w-md">
        <div class="p-6">
            <core:heading size="lg" class="mb-4">{{ __('agentic::agentic.session_detail.complete_session') }}</core:heading>
            <core:text class="mb-4">{{ __('agentic::agentic.session_detail.complete_session_prompt') }}</core:text>
            <core:textarea wire:model="completeSummary" placeholder="Session completed successfully..." rows="3" class="mb-4" />
            <div class="flex justify-end gap-2">
                <core:button wire:click="$set('showCompleteModal', false)" variant="ghost">{{ __('agentic::agentic.actions.cancel') }}</core:button>
                <core:button wire:click="completeSession" variant="primary">{{ __('agentic::agentic.actions.complete_session') }}</core:button>
            </div>
        </div>
    </core:modal>

    {{-- Fail Modal --}}
    <core:modal wire:model="showFailModal" class="max-w-md">
        <div class="p-6">
            <core:heading size="lg" class="mb-4">{{ __('agentic::agentic.session_detail.fail_session') }}</core:heading>
            <core:text class="mb-4">{{ __('agentic::agentic.session_detail.fail_session_prompt') }}</core:text>
            <core:textarea wire:model="failReason" placeholder="Session failed due to..." rows="3" class="mb-4" />
            <div class="flex justify-end gap-2">
                <core:button wire:click="$set('showFailModal', false)" variant="ghost">{{ __('agentic::agentic.actions.cancel') }}</core:button>
                <core:button wire:click="failSession" variant="danger">{{ __('agentic::agentic.actions.mark_as_failed') }}</core:button>
            </div>
        </div>
    </core:modal>

    {{-- Replay Modal --}}
    <core:modal wire:model="showReplayModal" class="max-w-lg">
        <div class="p-6">
            <core:heading size="lg" class="mb-4">{{ __('agentic::agentic.session_detail.replay_session') }}</core:heading>
            <core:text class="mb-4">{{ __('agentic::agentic.session_detail.replay_session_prompt') }}</core:text>

            {{-- Replay Context Summary --}}
            @if($showReplayModal)
                <div class="bg-zinc-50 dark:bg-zinc-800 rounded-lg p-4 mb-4 space-y-2">
                    <div class="flex justify-between text-sm">
                        <span class="text-zinc-500">{{ __('agentic::agentic.session_detail.total_actions') }}</span>
                        <span class="font-medium">{{ $this->replayContext['total_actions'] ?? 0 }}</span>
                    </div>
                    <div class="flex justify-between text-sm">
                        <span class="text-zinc-500">{{ __('agentic::agentic.session_detail.checkpoints') }}</span>
                        <span class="font-medium">{{ count($this->replayContext['checkpoints'] ?? []) }}</span>
                    </div>
                    @if(isset($this->replayContext['last_checkpoint']))
                        <div class="text-sm">
                            <span class="text-zinc-500">{{ __('agentic::agentic.session_detail.last_checkpoint') }}:</span>
                            <span class="font-medium">{{ $this->replayContext['last_checkpoint']['message'] ?? 'N/A' }}</span>
                        </div>
                    @endif
                </div>
            @endif

            <div class="mb-4">
                <core:label for="replayAgentType">{{ __('agentic::agentic.session_detail.agent_type') }}</core:label>
                <core:select wire:model="replayAgentType" id="replayAgentType">
                    <option value="opus">Opus</option>
                    <option value="sonnet">Sonnet</option>
                    <option value="haiku">Haiku</option>
                </core:select>
            </div>

            <div class="flex justify-end gap-2">
                <core:button wire:click="$set('showReplayModal', false)" variant="ghost">{{ __('agentic::agentic.actions.cancel') }}</core:button>
                <core:button wire:click="replaySession" variant="primary" icon="arrow-path">{{ __('agentic::agentic.actions.replay_session') }}</core:button>
            </div>
        </div>
    </core:modal>
</div>
