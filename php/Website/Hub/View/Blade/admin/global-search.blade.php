<div
    x-data="{ open: @entangle('open') }"
    x-show="open"
    x-cloak
    class="fixed inset-0 z-50 flex items-start justify-center bg-zinc-950/40 px-4 py-16"
>
    <div
        class="w-full max-w-2xl overflow-hidden rounded-2xl border border-zinc-200 bg-white shadow-2xl"
        @click.outside="$wire.closeSearch()"
        @keydown.escape.window="$wire.closeSearch()"
        @keydown.arrow-up.window.prevent="$wire.navigateUp()"
        @keydown.arrow-down.window.prevent="$wire.navigateDown()"
        @keydown.enter.window.prevent="$wire.selectCurrent()"
    >
        <div class="border-b border-zinc-200 p-4">
            <input
                wire:model.live.debounce.200ms="query"
                type="text"
                placeholder="Search the Hub"
                class="w-full border-0 p-0 text-base text-zinc-900 focus:outline-none focus:ring-0"
                autofocus
            />
        </div>

        @if(strlen($query) >= 2)
            <div class="max-h-[28rem] overflow-y-auto">
                @php $currentIndex = 0; @endphp
                @forelse($this->results as $group)
                    <div class="border-b border-zinc-100 px-4 py-2 text-xs font-semibold uppercase tracking-[0.2em] text-zinc-500">
                        {{ $group['label'] }}
                    </div>

                    @foreach($group['results'] as $result)
                        <button
                            type="button"
                            class="block w-full border-b border-zinc-100 px-4 py-3 text-left {{ $selectedIndex === $currentIndex ? 'bg-violet-50' : 'hover:bg-zinc-50' }}"
                            wire:click="navigateTo({{ json_encode($result) }})"
                        >
                            <div class="font-medium text-zinc-900">{{ $result['title'] }}</div>
                            @if(!empty($result['subtitle']))
                                <div class="text-sm text-zinc-600">{{ $result['subtitle'] }}</div>
                            @endif
                        </button>
                        @php $currentIndex++; @endphp
                    @endforeach
                @empty
                    <div class="p-6 text-sm text-zinc-500">No results for “{{ $query }}”.</div>
                @endforelse
            </div>
        @elseif($this->showRecentSearches)
            <div class="max-h-[24rem] overflow-y-auto">
                <div class="flex items-center justify-between border-b border-zinc-100 px-4 py-2 text-xs font-semibold uppercase tracking-[0.2em] text-zinc-500">
                    <span>Recent searches</span>
                    <button type="button" class="text-zinc-500 hover:text-zinc-900" wire:click="clearRecentSearches">Clear</button>
                </div>

                @foreach($recentSearches as $index => $recent)
                    <div class="flex items-center justify-between border-b border-zinc-100 px-4 py-3">
                        <button type="button" class="text-left" wire:click="navigateToRecent({{ $index }})">
                            <div class="font-medium text-zinc-900">{{ $recent['title'] }}</div>
                            @if(!empty($recent['subtitle']))
                                <div class="text-sm text-zinc-600">{{ $recent['subtitle'] }}</div>
                            @endif
                        </button>

                        <button type="button" class="text-sm text-zinc-500 hover:text-zinc-900" wire:click="removeRecentSearch({{ $index }})">Remove</button>
                    </div>
                @endforeach
            </div>
        @else
            <div class="p-6 text-sm text-zinc-500">Type at least two characters to search registered Hub providers.</div>
        @endif
    </div>
</div>
