{{-- SPDX-License-Identifier: EUPL-1.2 --}}

<div class="space-y-6">
    <flux:card class="space-y-4">
        <div class="flex items-start justify-between gap-4">
            <div>
                <flux:heading size="lg">Brain Explorer</flux:heading>
                <flux:text>Search the OpenBrain corpus, inspect recall results, and forget stale entries.</flux:text>
            </div>

            <flux:button type="button" variant="ghost" wire:click="refreshExplorer">
                Refresh
            </flux:button>
        </div>

        <form class="space-y-4" wire:submit="searchMemories">
            <div class="grid gap-4 xl:grid-cols-[minmax(0,2fr)_repeat(3,minmax(0,1fr))]">
                <div class="space-y-2">
                    <flux:text class="text-xs uppercase tracking-wide text-zinc-500">Search Query</flux:text>
                    <flux:input wire:model="query" placeholder="How does dispatch history work?" />
                </div>

                <div class="space-y-2">
                    <flux:text class="text-xs uppercase tracking-wide text-zinc-500">Type</flux:text>
                    <flux:select wire:model="typeFilter">
                        <option value="">All types</option>
                        @foreach ($this->memoryTypes as $type)
                            <option value="{{ $type }}">{{ ucfirst($type) }}</option>
                        @endforeach
                    </flux:select>
                </div>

                <div class="space-y-2">
                    <flux:text class="text-xs uppercase tracking-wide text-zinc-500">Project</flux:text>
                    <flux:input wire:model="projectFilter" placeholder="core-agent" />
                </div>

                <div class="space-y-2">
                    <flux:text class="text-xs uppercase tracking-wide text-zinc-500">Agent</flux:text>
                    <flux:select wire:model="agentFilter">
                        <option value="">All agents</option>
                        @foreach ($this->availableAgents as $agent)
                            <option value="{{ $agent }}">{{ $agent }}</option>
                        @endforeach
                    </flux:select>
                </div>
            </div>

            <div class="flex items-center justify-between gap-3">
                <div class="flex flex-wrap items-center gap-2">
                    <flux:badge color="{{ $usedFallbackSearch ? 'warning' : 'success' }}">
                        {{ $usedFallbackSearch ? 'DB fallback search' : 'Semantic recall' }}
                    </flux:badge>
                    <flux:text>{{ count($results) }} result{{ count($results) === 1 ? '' : 's' }}</flux:text>
                </div>

                <flux:button type="submit" variant="primary">
                    Search Brain
                </flux:button>
            </div>
        </form>
    </flux:card>

    <div class="space-y-4">
        @forelse ($results as $result)
            <flux:card class="space-y-4">
                <div class="flex flex-wrap items-start justify-between gap-4">
                    <div class="space-y-2">
                        <div class="flex flex-wrap items-center gap-2">
                            <flux:badge color="{{ $this->typeBadgeVariant($result['type']) }}">
                                {{ strtoupper($result['type']) }}
                            </flux:badge>

                            @if (! is_null($result['score']))
                                <flux:badge color="zinc">
                                    Score {{ number_format((float) $result['score'], 3) }}
                                </flux:badge>
                            @endif

                            <flux:badge color="zinc">
                                {{ $result['agent_id'] }}
                            </flux:badge>
                        </div>

                        <flux:text class="leading-7 text-zinc-800">
                            {{ $result['content'] }}
                        </flux:text>
                    </div>

                    <flux:button type="button" variant="danger" wire:click="forgetMemory('{{ $result['id'] }}')">
                        Forget
                    </flux:button>
                </div>

                <div class="flex flex-wrap items-center gap-2 text-xs text-zinc-500">
                    @if (! empty($result['project']))
                        <span>Project: {{ $result['project'] }}</span>
                    @endif

                    @if (! empty($result['org']))
                        <span>Organisation: {{ $result['org'] }}</span>
                    @endif

                    <span>Confidence: {{ number_format((float) $result['confidence'], 2) }}</span>

                    @if (! empty($result['created_at']))
                        <span>Created: {{ $result['created_at'] }}</span>
                    @endif
                </div>

                @if ($result['tags'] !== [])
                    <div class="flex flex-wrap gap-2">
                        @foreach ($result['tags'] as $tag)
                            <flux:badge color="zinc">{{ $tag }}</flux:badge>
                        @endforeach
                    </div>
                @endif
            </flux:card>
        @empty
            <flux:card>
                <flux:text>No memories found for the current search.</flux:text>
            </flux:card>
        @endforelse
    </div>
</div>
